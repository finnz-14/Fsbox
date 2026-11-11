// DevKit starter mod. Edit this file, then run build.bat to produce main.wasm.
// You can also drop this main.go into a mod folder under Mods/ and let the
// engine build it with the main build.bat.
package main

import (
	"fmt"

	fsbox "fsbox.dev/sdk"
)

func main() {
	fsbox.Log("DevKit mod loaded")
	fsbox.OnTick(func() {
		info, err := fsbox.GetEngineInfo()
		if err != nil {
			return
		}
		if info.State == "playing" {
			// Your per-frame logic goes here.
		}
	})

	fmt.Println("DevKit ready")
}
