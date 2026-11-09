package main

import (
	"fmt"

	fsbox "fsbox.dev/sdk"
)

var spawned bool

func init() {
	fsbox.On("game.start", func(e fsbox.Event) {
		fsbox.Log("examplemod ready")
	})
	fsbox.OnTick(func() {
		if spawned {
			return
		}
		spawned = true
		fsbox.Log("tick: spawning")
		id, err := fsbox.SpawnEntity("box", fsbox.Vec3{X: 0, Y: 4, Z: 0}, fsbox.Color{R: 100, G: 180, B: 220, A: 255})
		fsbox.Log(fmt.Sprintf("spawned entity %d err=%v", id, err))
	})
}

func main() {}
