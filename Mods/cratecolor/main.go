package main

import (
	fsbox "fsbox.dev/sdk"
)

func init() {
	fsbox.On("entity.grab", func(e fsbox.Event) {
		if e.Entity.Kind != "box" {
			return
		}
		fsbox.SetEntityColor(e.Entity.ID, fsbox.RandomColor())
		fsbox.Log("crate grabbed: " + e.Entity.Name)
	})
}

func main() {}
