package game

// Addon-facing networking helpers. The command router in modapi.go delegates
// here so the multiplayer code stays in one place.

// netModeName returns the stable string used by the addon API.
func (g *Game) netModeName() string {
	switch g.netMode {
	case netModeServer:
		return "server"
	case netModeClient:
		return "client"
	default:
		return "off"
	}
}

// netStatusInfo describes this process's multiplayer role. It is safe to call
// in single-player; the mode is then "off".
func (g *Game) netStatusInfo() NetStatusInfo {
	info := NetStatusInfo{
		Mode:     g.netModeName(),
		Server:   g.netMode == netModeServer,
		Map:      g.selectedMap,
		Protocol: netProtocolVersion,
	}
	switch g.netMode {
	case netModeServer:
		if g.netSrv != nil {
			info.Port = g.netSrv.port
			info.Players = len(g.netSrv.clients)
			info.Tick = g.netSrv.tick
		}
	case netModeClient:
		if g.netClient != nil {
			info.Connected = true
			info.Address = g.netClient.conn.RemoteAddr().String()
			info.PlayerID = g.netClient.id
			info.PlayerName = g.netClient.name
			info.Players = 1 + len(g.netRemote)
		}
	}
	return info
}

// netPlayerInfos lists every connected player. On a client the local player is
// included first so addons can reason about the full roster without special
// cases.
func (g *Game) netPlayerInfos() []NetPlayerInfo {
	switch g.netMode {
	case netModeServer:
		if g.netSrv == nil {
			return []NetPlayerInfo{}
		}
		out := make([]NetPlayerInfo, 0, len(g.netSrv.clients))
		for _, c := range g.netSrv.clients {
			if c.player == nil {
				continue
			}
			p := c.player.body.GetPosition()
			held := int32(0)
			if c.held != nil {
				held = c.held.ID
			}
			out = append(out, NetPlayerInfo{
				ID:       c.id,
				Name:     c.name,
				Position: Vec3Info{X: p.X, Y: p.Y, Z: p.Z},
				Yaw:      c.player.yaw,
				Pitch:    c.player.pitch,
				Fly:      c.player.fly,
				Held:     held,
			})
		}
		return out
	case netModeClient:
		out := make([]NetPlayerInfo, 0, 1+len(g.netRemote))
		if g.netClient != nil {
			p := g.player.body.GetPosition()
			held := int32(0)
			if g.held != nil {
				held = g.held.NetID
			}
			out = append(out, NetPlayerInfo{
				ID:       g.netClient.id,
				Name:     g.netClient.name,
				Position: Vec3Info{X: p.X, Y: p.Y, Z: p.Z},
				Yaw:      g.player.yaw,
				Pitch:    g.player.pitch,
				Fly:      g.player.fly,
				Held:     held,
				Local:    true,
			})
		}
		for _, rp := range g.netRemote {
			out = append(out, NetPlayerInfo{
				ID:       rp.ID,
				Name:     rp.Name,
				Position: Vec3Info{X: rp.Pos.X, Y: rp.Pos.Y, Z: rp.Pos.Z},
				Yaw:      rp.Yaw,
				Pitch:    rp.Pitch,
				Fly:      rp.Fly,
				Held:     rp.Held,
			})
		}
		return out
	default:
		return []NetPlayerInfo{}
	}
}
