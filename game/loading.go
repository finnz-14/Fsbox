package game

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// drawLoadingScreen paints a simple progress screen and pumps window events so
// the OS does not mark the window as "not responding" while startup (content
// loading, model import, addon loading) blocks the main thread.
func (g *Game) drawLoadingScreen(progress float32, text string) {
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}

	rl.PollInputEvents()
	rl.BeginDrawing()
	rl.ClearBackground(rl.Color{R: 7, G: 8, B: 10, A: 255})

	w := rl.GetScreenWidth()
	h := rl.GetScreenHeight()
	cx := float32(w) * 0.5
	cy := float32(h) * 0.5

	drawLoadingTextCentered(g, "FSBOX", cx, cy-90, 42, rl.Color{R: 240, G: 173, B: 78, A: 255})
	drawLoadingTextCentered(g, text, cx, cy-20, 16, rl.Color{R: 184, G: 192, B: 204, A: 255})

	barW := float32(w) * 0.36
	barH := float32(6)
	bx := cx - barW*0.5
	by := cy + 20
	rl.DrawRectangleRec(rl.Rectangle{X: bx, Y: by, Width: barW, Height: barH}, rl.Color{R: 30, G: 32, B: 38, A: 255})
	rl.DrawRectangleRec(rl.Rectangle{X: bx, Y: by, Width: barW * progress, Height: barH}, rl.Color{R: 240, G: 173, B: 78, A: 255})

	drawLoadingTextCentered(g, "LOADING", cx, by+18, 12, rl.Color{R: 122, G: 132, B: 148, A: 255})

	rl.EndDrawing()
}

func drawLoadingText(g *Game, text string, x, y float32, size int32, color rl.Color) {
	if g != nil && g.font.Texture.ID != 0 {
		rl.DrawTextEx(g.font, text, rl.Vector2{X: x, Y: y}, float32(size), 1.2, color)
		return
	}
	rl.DrawText(text, int32(x), int32(y), size, color)
}

func measureLoadingTextWidth(g *Game, text string, size int32) float32 {
	if g != nil && g.font.Texture.ID != 0 {
		return rl.MeasureTextEx(g.font, text, float32(size), 1.2).X
	}
	return float32(rl.MeasureText(text, size))
}

func drawLoadingTextCentered(g *Game, text string, centerX, y float32, size int32, color rl.Color) {
	drawLoadingText(g, text, centerX-measureLoadingTextWidth(g, text, size)*0.5, y, size, color)
}
