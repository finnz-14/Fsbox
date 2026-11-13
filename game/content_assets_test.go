package game

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// TestMain installs a synthetic fs.FS built from the repository root's
// Content/ folder as the embedded asset filesystem for headless tests and
// opens a hidden GL context, which is required by raylib to load map models.
func TestMain(m *testing.M) {
	wd, err := os.Getwd()
	if err == nil {
		root := filepath.Dir(wd)
		assets, err := mapFS(root)
		if err == nil {
			SetEmbeddedAssets(assets, assets)
		}
	}
	rl.SetConfigFlags(rl.FlagWindowHidden)
	rl.InitWindow(1, 1, "fsbox-tests")
	code := m.Run()
	rl.CloseWindow()
	os.Exit(code)
}

func mapFS(root string) (fs.FS, error) {
	base := filepath.Join(root, "Content")
	op := fstest.MapFS{}
	err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		data, _ := os.ReadFile(path)
		if d.IsDir() {
			data = nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			op[rel] = &fstest.MapFile{Mode: os.ModeDir | 0755}
			return nil
		}
		op[rel] = &fstest.MapFile{Data: data, Mode: 0644}
		return nil
	})
	return op, err
}
