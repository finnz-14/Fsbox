package game

import (
	"encoding/json"
	"net"
	"os"
	"strconv"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func (g *Game) netAddrRect() rl.Rectangle {
	rec := g.mainMenuPanel()
	return rl.Rectangle{X: rec.X + 30, Y: rec.Y + 258, Width: rec.Width - 60, Height: 34}
}

func (g *Game) netNameRect() rl.Rectangle {
	rec := g.mainMenuPanel()
	return rl.Rectangle{X: rec.X + 30, Y: rec.Y + 318, Width: rec.Width - 60, Height: 34}
}

func (g *Game) netBackRect() rl.Rectangle {
	rec := g.mainMenuPanel()
	return rl.Rectangle{X: rec.X + 30, Y: rec.Y + 472, Width: rec.Width - 60, Height: 34}
}

func (g *Game) netListRect() rl.Rectangle {
	rec := g.mainMenuPanel()
	return rl.Rectangle{X: rec.X + 30, Y: rec.Y + 92, Width: rec.Width - 60, Height: 148}
}

func (g *Game) netListItemRect(i int) rl.Rectangle {
	lr := g.netListRect()
	return rl.Rectangle{X: lr.X + 4, Y: lr.Y + 4 + float32(i)*22, Width: lr.Width - 8, Height: 20}
}

func (g *Game) netAddRect() rl.Rectangle {
	rec := g.mainMenuPanel()
	half := (rec.Width - 66) * 0.5
	return rl.Rectangle{X: rec.X + 30, Y: rec.Y + 378, Width: half, Height: 34}
}

func (g *Game) netRemoveRect() rl.Rectangle {
	rec := g.mainMenuPanel()
	half := (rec.Width - 66) * 0.5
	return rl.Rectangle{X: rec.X + 36 + half, Y: rec.Y + 378, Width: half, Height: 34}
}

func (g *Game) netConnectRect() rl.Rectangle {
	rec := g.mainMenuPanel()
	return rl.Rectangle{X: rec.X + 30, Y: rec.Y + 420, Width: rec.Width - 60, Height: 40}
}

func loadServerList() []string {
	data, err := os.ReadFile("servers.json")
	if err != nil {
		return []string{}
	}
	var out []string
	_ = json.Unmarshal(data, &out)
	// Drop empties and de-duplicate while preserving order.
	seen := make(map[string]bool)
	clean := make([]string, 0, len(out))
	for _, s := range out {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		clean = append(clean, s)
	}
	return clean
}

func saveServerList(list []string) {
	data, _ := json.MarshalIndent(list, "", "  ")
	_ = os.WriteFile("servers.json", data, 0644)
}

func (g *Game) updateNetMenuInput() {
	// A background dial reports back here; the main thread applies the world
	// reset so no box3d/raylib state is touched from the worker goroutine.
	select {
	case res := <-g.netConnectCh:
		g.netConnecting = false
		if res.err != nil {
			g.netStatus = "FAILED // " + strings.ToUpper(res.err.Error())
			g.netStatusAt = colDanger
		} else if res.client != nil {
			g.startNetClient(res.client)
			return
		}
	default:
	}

	if g.netConnecting {
		return
	}

	mouse := g.uiMouse()
	if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
		// Server list selection.
		for i := range g.netServerList {
			if rl.CheckCollisionPointRec(mouse, g.netListItemRect(i)) {
				g.netServerSel = i
				g.netAddr = g.netServerList[i]
				g.netField = 0
				return
			}
		}
		switch {
		case rl.CheckCollisionPointRec(mouse, g.netAddrRect()):
			g.netField = 1
			return
		case rl.CheckCollisionPointRec(mouse, g.netNameRect()):
			g.netField = 2
			return
		case rl.CheckCollisionPointRec(mouse, g.netAddRect()):
			addr := strings.TrimSpace(g.netAddr)
			if addr != "" {
				found := false
				for _, s := range g.netServerList {
					if s == addr {
						found = true
						break
					}
				}
				if !found {
					g.netServerList = append(g.netServerList, addr)
					saveServerList(g.netServerList)
				}
				for i, s := range g.netServerList {
					if s == addr {
						g.netServerSel = i
						break
					}
				}
			}
			return
		case rl.CheckCollisionPointRec(mouse, g.netRemoveRect()):
			if g.netServerSel >= 0 && g.netServerSel < len(g.netServerList) {
				g.netServerList = append(g.netServerList[:g.netServerSel], g.netServerList[g.netServerSel+1:]...)
				saveServerList(g.netServerList)
				if g.netServerSel >= len(g.netServerList) {
					g.netServerSel = len(g.netServerList) - 1
				}
				if g.netServerSel >= 0 && g.netServerSel < len(g.netServerList) {
					g.netAddr = g.netServerList[g.netServerSel]
				}
			}
			return
		case rl.CheckCollisionPointRec(mouse, g.netConnectRect()):
			g.beginNetConnect()
			return
		case rl.CheckCollisionPointRec(mouse, g.netBackRect()):
			g.state = stateMainMenu
			return
		default:
			g.netField = 0
		}
	}

	if g.netField == 0 {
		return
	}
	for {
		ch := rl.GetCharPressed()
		if ch == 0 {
			break
		}
		if ch >= 32 && ch < 127 {
			switch g.netField {
			case 1:
				if len(g.netAddr) < 64 {
					g.netAddr += string(rune(ch))
				}
			case 2:
				if len(g.netName) < netMaxNameLen {
					g.netName += string(rune(ch))
				}
			}
		}
	}
	if rl.IsKeyPressed(rl.KeyBackspace) {
		switch g.netField {
		case 1:
			if len(g.netAddr) > 0 {
				g.netAddr = g.netAddr[:len(g.netAddr)-1]
			}
		case 2:
			if len(g.netName) > 0 {
				g.netName = g.netName[:len(g.netName)-1]
			}
		}
	}
	if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter) {
		g.beginNetConnect()
	}
}

// normalizeNetAddr accepts "host", "host:port" or a bare IPv6 literal and
// returns a dialable host:port string.
func normalizeNetAddr(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		addr = "127.0.0.1"
	}
	if _, _, err := net.SplitHostPort(addr); err == nil {
		return addr
	}
	// Bare IPv6 literals contain multiple colons; JoinHostPort adds brackets.
	if strings.Count(addr, ":") > 1 {
		return net.JoinHostPort(addr, strconv.Itoa(netDefaultPort))
	}
	if !strings.Contains(addr, ":") {
		return net.JoinHostPort(addr, strconv.Itoa(netDefaultPort))
	}
	return addr
}

func (g *Game) beginNetConnect() {
	if g.netConnecting {
		return
	}
	addr := normalizeNetAddr(g.netAddr)
	name := sanitizeNetName(strings.TrimSpace(g.netName))
	g.netConnecting = true
	g.netField = 0
	g.netStatus = "CONNECTING TO " + addr + " ..."
	g.netStatusAt = colAccent

	ch := g.netConnectCh
	go func() {
		c, err := dialNetClient(addr, name)
		ch <- netConnectResult{client: c, err: err}
	}()
}

func (g *Game) drawNetMenu() {
	w := g.uiWidth()
	h := g.uiHeight()
	rl.DrawRectangle(0, 0, int32(w), int32(h), rgba(4, 5, 6, 232))

	rec := g.mainMenuPanel()
	g.panel(rec, colPanelDeep, colBorderHi)
	g.accentBar(rl.Rectangle{X: rec.X, Y: rec.Y, Width: 3, Height: rec.Height}, colAccent)

	g.textCenteredSpaced("MULTIPLAYER", rec.X+rec.Width*0.5, rec.Y+26, 26, 4.0, colText)
	g.textCenteredSpaced("DEDICATED SERVER LINK", rec.X+rec.Width*0.5, rec.Y+62, 10, 1.8, colMuted)

	// Server list.
	lr := g.netListRect()
	g.panel(lr, colPanelSoft, colBorderHi)
	if len(g.netServerList) == 0 {
		g.textCenteredSpaced("NO SAVED SERVERS", lr.X+lr.Width*0.5, lr.Y+lr.Height*0.5-6, 9, 1.2, colMuted)
	}
	for i, addr := range g.netServerList {
		ir := g.netListItemRect(i)
		fill := colPanelSoft
		border := colBorder
		if i == g.netServerSel {
			fill = colPanelHover
			border = colAccent
		}
		rl.DrawRectangleRec(ir, fill)
		rl.DrawRectangleLinesEx(ir, 1, border)
		g.drawTextSpaced(addr, ir.X+8, ir.Y+4, 12, 0.6, colText)
	}

	g.drawSectionLabel("SERVER ADDRESS", rec.X+30, rec.Y+248, colAccent)
	g.drawNetField(g.netAddrRect(), g.netAddr, g.netField == 1, "127.0.0.1:27015")

	g.drawSectionLabel("CALLSIGN", rec.X+30, rec.Y+308, colAccent)
	g.drawNetField(g.netNameRect(), g.netName, g.netField == 2, "PLAYER")

	g.drawMenuButton(g.netAddRect(), "ADD SERVER", false)
	g.drawMenuButton(g.netRemoveRect(), "REMOVE", true)

	label := "CONNECT"
	if g.netConnecting {
		label = "CONNECTING..."
	}
	g.drawMenuButton(g.netConnectRect(), label, false)
	g.drawMenuButton(g.netBackRect(), "BACK", true)

	status := g.netStatus
	if status == "" {
		status = "SELECT OR ENTER A SERVER ADDRESS"
	}
	g.textCenteredSpaced(status, rec.X+rec.Width*0.5, rec.Y+362, 9.5, 1.2, g.netStatusAt)
}

func (g *Game) drawNetField(rec rl.Rectangle, value string, focused bool, placeholder string) {
	fill := colPanelSoft
	border := colBorderHi
	if focused {
		fill = colPanelHover
		border = colAccent
	}
	rl.DrawRectangleRec(rec, fill)
	rl.DrawRectangleLinesEx(rec, 1, border)
	rl.DrawRectangle(int32(rec.X+6), int32(rec.Y+rec.Height*0.5-1), 8, 2, border)

	text := value
	textCol := colText
	if text == "" {
		text = placeholder
		textCol = withAlpha(colMuted, 160)
	}
	g.drawTextSpaced(text, rec.X+24, rec.Y+(rec.Height-13)*0.5, 13, 0.8, textCol)
	if focused && (int(g.uiTime*2)%2 == 0) {
		tw := g.textWidthSpaced(text, 13, 0.8)
		rl.DrawRectangle(int32(rec.X+24+tw+2), int32(rec.Y+10), 2, int32(rec.Height-20), colAccent)
	}
}
