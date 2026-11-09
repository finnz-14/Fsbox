package main

import (
	"math"
	"unsafe"

	box3d "github.com/gataengine/box3d-go"
	rl "github.com/gen2brain/raylib-go/raylib"
)

func (g *Game) Draw() {
	rl.BeginDrawing()

	// Clear both colour and depth. The sky gradient is painted afterwards, so
	// this only exists to reset the depth buffer for the next 3D pass.
	rl.ClearBackground(g.skyBottom)
	g.drawBackground()

	if g.hasShadow && g.settings.Shadows {
		d := normRl(g.sunDir)
	g.sunCam.Position = rlv(d.X*40, d.Y*40, d.Z*40)
		g.sunCam.Target = rlv(0, 0, 0)
		g.sunCam.Up = rlv(0, 1, 0)
		g.sunCam.Fovy = 80
		g.sunCam.Projection = rl.CameraOrthographic

		rl.BeginTextureMode(g.shadowMap)
		rl.ClearBackground(rl.White)
		rl.BeginMode3D(g.sunCam)
		// Use the unified shader in shadow-pass mode so alpha-masked texels
		// do not cast shadows.
		rl.BeginShaderMode(g.mainShader)
		locShadowPass := rl.GetShaderLocation(g.mainShader, "shadowPass")
		locShadowEnabled := rl.GetShaderLocation(g.mainShader, "shadowEnabled")
		sp := int32(1)
		se := int32(0)
		rl.SetShaderValue(g.mainShader, locShadowPass, int32Slice(sp), rl.ShaderUniformInt)
		rl.SetShaderValue(g.mainShader, locShadowEnabled, int32Slice(se), rl.ShaderUniformInt)
		g.drawEntities()
		rl.EndShaderMode()
		g.lightVP = rl.MatrixMultiply(rl.GetMatrixProjection(), rl.GetMatrixModelview())
		rl.EndMode3D()
		rl.EndTextureMode()
	}

	rl.BeginMode3D(g.shakeCamera())
	if g.hasShadow {
		rl.BeginShaderMode(g.mainShader)
		locLightVP := rl.GetShaderLocation(g.mainShader, "lightVP")
		locLightDir := rl.GetShaderLocation(g.mainShader, "lightDir")
		locShadowMap := rl.GetShaderLocation(g.mainShader, "shadowMap")
		locShadowPass := rl.GetShaderLocation(g.mainShader, "shadowPass")
		locShadowEnabled := rl.GetShaderLocation(g.mainShader, "shadowEnabled")
		lightVP := g.lightVP
		if !g.settings.Shadows {
			lightVP = rl.MatrixIdentity()
		}
		d := normRl(g.sunDir)
		lightDir := rlv(-d.X, -d.Y, -d.Z)
		rl.SetShaderValueMatrix(g.mainShader, locLightVP, lightVP)
		rl.SetShaderValue(g.mainShader, locLightDir, []float32{lightDir.X, lightDir.Y, lightDir.Z}, rl.ShaderUniformVec3)
		sp := int32(0)
		rl.SetShaderValue(g.mainShader, locShadowPass, int32Slice(sp), rl.ShaderUniformInt)
		if g.settings.Shadows {
			se := int32(1)
			rl.SetShaderValue(g.mainShader, locShadowEnabled, int32Slice(se), rl.ShaderUniformInt)
			// Bind shadow map manually to slot 7.
			rl.ActiveTextureSlot(7)
			rl.EnableTexture(g.shadowMap.Texture.ID)
			slot := int32(7)
			rl.SetShaderValue(g.mainShader, locShadowMap, int32Slice(slot), rl.ShaderUniformInt)
		} else {
			se := int32(0)
			rl.SetShaderValue(g.mainShader, locShadowEnabled, int32Slice(se), rl.ShaderUniformInt)
		}
	}
	g.drawEntities()
	if g.hasShadow {
		rl.EndShaderMode()
	}
	g.drawSun()
	g.drawAimHighlight()
	g.drawGhost()
	g.drawHeldBeam()
	g.drawWaves()
	g.drawParticles()
	g.drawModParticles()
	g.drawModDebug()
	rl.EndMode3D()

	// The tool gun is a separate, close-range 3D pass. Drawing it after the
	// world means its depth values win over almost everything else.
	rl.BeginMode3D(g.camera)
	g.drawViewmodel()
	rl.EndMode3D()

	g.drawVignette()

	uiCam := rl.Camera2D{Zoom: g.uiScale()}
	rl.BeginMode2D(uiCam)
	g.drawHUD()
	switch g.state {
	case stateHelp:
		g.drawHelp()
	case statePaused:
		g.drawPause()
	case stateSettings:
		g.drawSettings()
	case stateSpawn:
		g.drawSpawnMenu()
	case stateToolMenu:
		g.drawToolMenu()
	}
	g.drawToasts()
	g.drawModUI()
	rl.EndMode2D()

	rl.EndDrawing()
}

// ---------------------------------------------------------------------------
// Background and atmosphere
// ---------------------------------------------------------------------------

func (g *Game) drawBackground() {
	w := int32(rl.GetScreenWidth())
	h := int32(rl.GetScreenHeight())
	rl.DrawRectangleGradientV(0, 0, w, h, g.skyTop, g.skyBottom)
}

func (g *Game) drawVignette() {
	w := float32(rl.GetScreenWidth())
	h := float32(rl.GetScreenHeight())
	clear := rgba(0, 0, 0, 0)
	edge := rgba(0, 0, 0, 80)

	rl.DrawRectangleGradientEx(rl.Rectangle{X: 0, Y: 0, Width: w, Height: 130}, edge, clear, clear, edge)
	rl.DrawRectangleGradientEx(rl.Rectangle{X: 0, Y: h - 150, Width: w, Height: 150}, clear, edge, edge, clear)
	rl.DrawRectangleGradientEx(rl.Rectangle{X: 0, Y: 0, Width: 120, Height: h}, edge, edge, clear, clear)
	rl.DrawRectangleGradientEx(rl.Rectangle{X: w - 120, Y: 0, Width: 120, Height: h}, clear, clear, edge, edge)
}

func (g *Game) drawArenaGrid() {
	const y = 0.012
	minor := rgba(150, 160, 170, 34)
	major := rgba(196, 172, 120, 76)
	for i := -30; i <= 30; i++ {
		c := minor
		if i%5 == 0 {
			c = major
		}
		x := float32(i)
		rl.DrawLine3D(rlv(x, y, -30), rlv(x, y, 30), c)
		rl.DrawLine3D(rlv(-30, y, x), rlv(30, y, x), c)
	}
}

func int32Slice(v int32) []float32 {
	// rl.SetShaderValue takes []float32 but reads ints when the uniform type
	// is ShaderUniformInt. Hand it a slice whose backing memory is an int32.
	arr := [1]int32{v}
	return (*[1]float32)(unsafe.Pointer(&arr))[:]
}

func (g *Game) drawSun() {
	// Visible sun along the light direction used for shading/shadows.
	dir := g.sunDir
	l := lenRl(dir)
	if l < 1e-6 {
		return
	}
	dir = rlv(dir.X/l, dir.Y/l, dir.Z/l)
	pos := rlv(dir.X*60, dir.Y*60, dir.Z*60)
	rl.DrawSphereEx(pos, 2.2, 12, 16, rl.NewColor(255, 240, 180, 255))
	rl.DrawSphereWires(pos, 2.4, 8, 12, rl.NewColor(255, 220, 120, 180))
}

// ---------------------------------------------------------------------------
// Entities
// ---------------------------------------------------------------------------

func (g *Game) drawEntities() {
	cam := g.camera.Position
	for _, e := range g.entities {
		p := e.position()
		if vlen(p.X-cam.X, p.Y-cam.Y, p.Z-cam.Z) > 95 {
			continue
		}
		q := e.Body.GetRotation()
		c := e.Color
		edge := colEdge
		if e.Static {
			edge = rgba(8, 16, 18, 95)
		}
		if e.Frozen {
			edge = withAlpha(colFreeze, 205)
		}
		g.drawEntityShape(e, p, q, c, edge)
	}
}

func (g *Game) drawEntityShape(e *Entity, p rl.Vector3, q box3d.Quat, c, edge rl.Color) {
	if e.HasModel {
		axis, angle := quatAxisAngle(q)
		scale := rlv(1, 1, 1)
		// Content models are authored double-sided (glTF doubleSided=true), so
		// keep both faces visible instead of letting the default culling carve
		// holes out of the mesh.
		rl.DisableBackfaceCulling()
		rl.DrawModelEx(e.Model, p, axis, angle, scale, c)
		rl.EnableBackfaceCulling()
		return
	}
	switch e.Kind {
	case PropBox:
		axis, angle := quatAxisAngle(q)
		scale := rlv(e.Half.X*2, e.Half.Y*2, e.Half.Z*2)
		rl.DrawModelEx(g.cube, p, axis, angle, scale, c)
		if e.Name != "Ground" {
			drawBoxEdges(p, q, e.Half, edge)
		}
	case PropSphere:
		rl.DrawSphereEx(p, e.Radius, 14, 20, c)
		// A plain shaded sphere hides its rotation completely. Two great
		// circles plus a surface dot make rolling and spinning obvious.
		qRl := rlQuat(q)
		ax := rl.Vector3RotateByQuaternion(rlv(1, 0, 0), qRl)
		ay := rl.Vector3RotateByQuaternion(rlv(0, 1, 0), qRl)
		drawGreatCircle(p, ax, e.Radius*1.003, withAlpha(edge, 170))
		drawGreatCircle(p, ay, e.Radius*1.003, withAlpha(edge, 120))
		marker := rlv(p.X+ax.X*e.Radius*0.88, p.Y+ax.Y*e.Radius*0.88, p.Z+ax.Z*e.Radius*0.88)
		rl.DrawSphereEx(marker, maxf(0.045, e.Radius*0.15), 5, 7, withAlpha(rgb(255, 255, 255), 150))
	case PropCylinder:
		qRl := rlQuat(q)
		axis := rl.Vector3RotateByQuaternion(rlv(0, 1, 0), qRl)
		half := e.Height * 0.5
		start := rlv(p.X-axis.X*half, p.Y-axis.Y*half, p.Z-axis.Z*half)
		end := rlv(p.X+axis.X*half, p.Y+axis.Y*half, p.Z+axis.Z*half)
		rl.DrawCylinderEx(start, end, e.Radius, e.Radius, int32(e.Sides), c)
		rl.DrawCylinderWiresEx(start, end, e.Radius, e.Radius, int32(e.Sides), withAlpha(edge, 150))
	}
}

func drawBoxEdges(p rl.Vector3, q box3d.Quat, half box3d.Vec3, c rl.Color) {
	qRl := rlQuat(q)
	var corners [8]rl.Vector3
	for i := 0; i < 8; i++ {
		sx := float32(-1)
		if i&1 != 0 {
			sx = 1
		}
		sy := float32(-1)
		if i&2 != 0 {
			sy = 1
		}
		sz := float32(-1)
		if i&4 != 0 {
			sz = 1
		}
		local := rlv(sx*half.X, sy*half.Y, sz*half.Z)
		rot := rl.Vector3RotateByQuaternion(local, qRl)
		corners[i] = rlv(p.X+rot.X, p.Y+rot.Y, p.Z+rot.Z)
	}
	for i := 0; i < 8; i++ {
		for bit := 1; bit <= 4; bit <<= 1 {
			if i&bit == 0 {
				rl.DrawLine3D(corners[i], corners[i|bit], c)
			}
		}
	}
}

func drawGreatCircle(center, normal rl.Vector3, radius float32, c rl.Color) {
	n := normRl(normal)
	helper := rlv(0, 1, 0)
	if absf(n.Y) > 0.92 {
		helper = rlv(1, 0, 0)
	}
	cross := func(a, b rl.Vector3) rl.Vector3 {
		return rlv(a.Y*b.Z-a.Z*b.Y, a.Z*b.X-a.X*b.Z, a.X*b.Y-a.Y*b.X)
	}
	u := normRl(cross(n, helper))
	v := cross(n, u)

	const segments = 28
	prev := rlv(center.X+u.X*radius, center.Y+u.Y*radius, center.Z+u.Z*radius)
	for i := 1; i <= segments; i++ {
		a := float64(i) / segments * math.Pi * 2
		ca := float32(math.Cos(a))
		sa := float32(math.Sin(a))
		pt := rlv(
			center.X+(u.X*ca+v.X*sa)*radius,
			center.Y+(u.Y*ca+v.Y*sa)*radius,
			center.Z+(u.Z*ca+v.Z*sa)*radius,
		)
		rl.DrawLine3D(prev, pt, c)
		prev = pt
	}
}

func (g *Game) drawGhost() {
	if !g.ghostOK || g.state != statePlaying || g.held != nil || g.tool != toolSpawner {
		return
	}
	p := presets[g.selected]
	pulse := 1 + 0.018*float32(math.Sin(g.time*5.5))
	pos := g.ghostPos
	col := withAlpha(p.Color, 105)
	outline := withAlpha(colAccent, 200)

	switch p.Kind {
	case PropBox:
		if p.HasModel {
			rl.DisableBackfaceCulling()
			rl.DrawModelEx(p.Model, pos, rlv(0, 1, 0), 0, rlv(pulse, pulse, pulse), col)
			rl.EnableBackfaceCulling()
		} else {
			scale := rlv(p.Half.X*2*pulse, p.Half.Y*2*pulse, p.Half.Z*2*pulse)
			rl.DrawModelEx(g.cube, pos, rlv(0, 1, 0), 0, scale, col)
			drawBoxEdges(pos, box3d.QuatIdentity, p.Half, outline)
		}
	case PropSphere:
		rl.DrawSphereEx(pos, p.Radius*pulse, 12, 16, col)
		rl.DrawSphereWires(pos, p.Radius*pulse, 7, 10, outline)
	case PropCylinder:
		h := p.Height * 0.5 * pulse
		rl.DrawCylinderEx(rlv(pos.X, pos.Y-h, pos.Z), rlv(pos.X, pos.Y+h, pos.Z), p.Radius*pulse, p.Radius*pulse, 16, col)
		rl.DrawCylinderWiresEx(rlv(pos.X, pos.Y-h, pos.Z), rlv(pos.X, pos.Y+h, pos.Z), p.Radius*pulse, p.Radius*pulse, 16, outline)
	}

	// Grounding tick under the ghost.
	rl.DrawLine3D(rlv(pos.X, pos.Y-0.5, pos.Z), rlv(pos.X, 0.04, pos.Z), rgba(230, 166, 66, 45))
}

func (g *Game) drawHeldBeam() {
	if g.held == nil || g.tool != toolPhysgun {
		return
	}
	e := g.held
	from := g.muzzleWorld()
	to := e.position()
	if g.held != nil {
		q := e.Body.GetRotation()
		off := rl.Vector3RotateByQuaternion(toRlV(g.holdOffset), rlQuat(q))
		to = rlv(to.X+off.X, to.Y+off.Y, to.Z+off.Z)
	}

	rl.DrawCylinderEx(from, to, 0.012, 0.012, 6, withAlpha(colAccent, 235))
	rl.DrawCylinderEx(from, to, 0.040, 0.040, 6, rgba(230, 166, 66, 34))

	for i := 0; i < 4; i++ {
		t := float32(math.Mod(g.uiTime*1.8+float64(i)*0.25, 1.0))
		px := from.X + (to.X-from.X)*t
		py := from.Y + (to.Y-from.Y)*t
		pz := from.Z + (to.Z-from.Z)*t
		rl.DrawSphereEx(rlv(px, py, pz), 0.045, 5, 7, withAlpha(colAccent, 210))
	}

	pulse := uint8(170 + 70*math.Sin(g.uiTime*8))
	g.drawEntityOutline(e, withAlpha(colAccent, pulse))
}

func (g *Game) drawEntityOutline(e *Entity, c rl.Color) {
	p := e.position()
	q := e.Body.GetRotation()
	switch e.Kind {
	case PropBox:
		drawBoxEdges(p, q, e.Half, c)
	case PropSphere:
		qRl := rlQuat(q)
		ax := rl.Vector3RotateByQuaternion(rlv(1, 0, 0), qRl)
		ay := rl.Vector3RotateByQuaternion(rlv(0, 1, 0), qRl)
		drawGreatCircle(p, ax, e.Radius*1.008, c)
		drawGreatCircle(p, ay, e.Radius*1.008, c)
	case PropCylinder:
		qRl := rlQuat(q)
		axis := rl.Vector3RotateByQuaternion(rlv(0, 1, 0), qRl)
		half := e.Height * 0.5
		start := rlv(p.X-axis.X*half, p.Y-axis.Y*half, p.Z-axis.Z*half)
		end := rlv(p.X+axis.X*half, p.Y+axis.Y*half, p.Z+axis.Z*half)
		rl.DrawCylinderWiresEx(start, end, e.Radius*1.01, e.Radius*1.01, int32(e.Sides), c)
	}
}

// drawAimHighlight is called from the HUD phase so it is not drawn twice by
// the held-object beam.
func (g *Game) drawAimHighlight() {
	if g.aim == nil || g.aim == g.held || g.state != statePlaying {
		return
	}
	pulse := uint8(145 + 65*math.Sin(g.uiTime*7))
	g.drawEntityOutline(g.aim, withAlpha(g.toolColor(), pulse))
}

func (g *Game) initShadows() {
	g.shadowMap = rl.LoadRenderTexture(1024, 1024)
	g.sunCam = rl.Camera3D{
		Position:   rlv(0, 40, 0),
		Target:     rlv(0, 0, 0),
		Up:         rlv(0, 1, 0),
		Fovy:       60,
		Projection: rl.CameraOrthographic,
	}
	mainVS := `#version 330
in vec3 vertexPosition;
in vec2 vertexTexCoord;
in vec3 vertexNormal;
in vec4 vertexColor;
uniform mat4 mvp;
uniform mat4 matModel;
uniform mat4 lightVP;
out vec2 fragTexCoord;
out vec4 fragColor;
out vec3 fragNormal;
out vec4 fragPosLight;
void main() {
    fragTexCoord = vertexTexCoord;
    fragColor = vertexColor;
    fragNormal = normalize(mat3(matModel) * vertexNormal);
    gl_Position = mvp * vec4(vertexPosition, 1.0);
    fragPosLight = lightVP * matModel * vec4(vertexPosition, 1.0);
}
`
	mainFS := `#version 330
in vec2 fragTexCoord;
in vec4 fragColor;
in vec3 fragNormal;
in vec4 fragPosLight;
uniform sampler2D texture0;
uniform vec4 colDiffuse;
uniform vec3 lightDir;
uniform sampler2D shadowMap;
uniform int shadowPass;
uniform int shadowEnabled;
out vec4 finalColor;
void main() {
    vec4 texelColor = texture(texture0, fragTexCoord);
    if (texelColor.a < 0.05) discard;
    if (shadowPass == 1) {
        finalColor = vec4(vec3(gl_FragCoord.z), 1.0);
        return;
    }
    vec3 projCoords = fragPosLight.xyz / fragPosLight.w;
    projCoords = projCoords * 0.5 + 0.5;
    float shadow = 0.0;
    if (shadowEnabled == 1 && projCoords.x >= 0.0 && projCoords.x <= 1.0 && projCoords.y >= 0.0 && projCoords.y <= 1.0 && projCoords.z <= 1.0) {
        float bias = 0.005;
        float closestDepth = texture(shadowMap, projCoords.xy).r;
        shadow = (projCoords.z - bias > closestDepth) ? 1.0 : 0.0;
    }
    vec3 n = fragNormal;
    if (length(n) < 0.1) n = vec3(0.0, 1.0, 0.0);
    n = normalize(n);
    float diff = max(dot(n, -lightDir), 0.0);
    vec3 base = texelColor.rgb * colDiffuse.rgb * fragColor.rgb;
    if (length(base) < 0.001) base = colDiffuse.rgb;
    vec3 color = base * (0.35 + 0.65 * diff * (1.0 - shadow));
    finalColor = vec4(color, texelColor.a * colDiffuse.a * fragColor.a);
}
`
	g.mainShader = rl.LoadShaderFromMemory(mainVS, mainFS)
	if g.cube.MaterialCount > 0 && g.mainShader.ID != 0 {
		mat := (*rl.Material)(g.cube.Materials)
		mat.Shader = g.mainShader
	}
	for i := range presets {
		if !presets[i].HasModel {
			continue
		}
		m := presets[i].Model
		for j := 0; j < int(m.MaterialCount); j++ {
			modelMaterial(&m, j).Shader = g.mainShader
		}
		presets[i].Model = m
	}
	g.hasShadow = g.mainShader.ID != 0 && g.shadowMap.ID != 0
}
func (g *Game) drawWaves() {
	for _, w := range g.waves {
		fade := clampf(w.Life/w.MaxLife, 0, 1)
		c := withAlpha(w.Color, uint8(190*fade))
		rl.DrawCircle3D(w.Center, w.Radius, rlv(1, 0, 0), 90, c)
		rl.DrawCircle3D(w.Center, w.Radius*0.84, rlv(1, 0, 0), 90, withAlpha(w.Color, uint8(105*fade)))
		rl.DrawSphereWires(w.Center, w.Radius*0.72, 7, 14, withAlpha(w.Color, uint8(55*fade)))
	}
}

func (g *Game) drawParticles() {
	for _, p := range g.particles {
		a := clampf(p.Life/p.MaxLife, 0, 1)
		size := p.Size * (0.45 + 0.55*a)
		rl.DrawSphereEx(p.Pos, size, 4, 6, withAlpha(p.Color, uint8(255*a)))
	}
}

// ---------------------------------------------------------------------------
// Camera shake
// ---------------------------------------------------------------------------

func (g *Game) shakeCamera() rl.Camera3D {
	cam := g.camera
	if g.shake <= 0.001 {
		return cam
	}
	amp := float32(g.shake) * 0.24
	cam.Position.X += (g.rng.Float32() - 0.5) * amp
	cam.Position.Y += (g.rng.Float32() - 0.5) * amp
	cam.Position.Z += (g.rng.Float32() - 0.5) * amp
	cam.Target.X += (g.rng.Float32() - 0.5) * amp * 0.35
	cam.Target.Y += (g.rng.Float32() - 0.5) * amp * 0.35
	cam.Target.Z += (g.rng.Float32() - 0.5) * amp * 0.35
	return cam
}
