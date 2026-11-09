package main

import (
	fsbox "fsbox.dev/sdk"
)

var (
	jennyModel  int32
	jennyEntity int32
	spawned     bool
)

func init() {
	fsbox.On("game.start", func(e fsbox.Event) {
		fsbox.Log("[Jenny] mod ready. Jenny will spawn on first tick.")
	})

	fsbox.OnTick(func() {
		if spawned {
			return
		}
		spawned = true

		// Load Jenny's model from the Content folder.
		model, err := fsbox.LoadModel("Content/Spawnables/Jenny/Jenny/jennynude.gltf")
		if err != nil {
			fsbox.Log("[Jenny] failed to load model: " + err.Error())
			return
		}
		jennyModel = model

		// Spawn Jenny near the player.
		id, err := fsbox.SpawnModelEntity(model, fsbox.Vec3{X: 0, Y: 2, Z: 4}, fsbox.Color{R: 255, G: 255, B: 255, A: 255})
		if err != nil {
			fsbox.Log("[Jenny] failed to spawn: " + err.Error())
			return
		}
		jennyEntity = id

		fsbox.Log("[Jenny] spawned. Walk up to her to see the animation GUI.")
		fsbox.Log("[Jenny] Note: this glTF uses node animations; raylib extracts 0 skeletal animations, so the animation GUI will be empty.")
	})
}

func main() {}
