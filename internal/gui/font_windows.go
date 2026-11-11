//go:build windows

package gui

import (
	"os"
	"path/filepath"
	"sync"

	"fyne.io/fyne/v2"
)

// systemFont retains the Windows reference interface's native Segoe UI and
// Cascadia/Consolas typography, falling back to Fyne if a font is missing.
var (
	fontOnce  sync.Once
	fontCache map[string]fyne.Resource
)

func systemFont(style fyne.TextStyle) fyne.Resource {
	fontOnce.Do(func() {
		fontCache = map[string]fyne.Resource{}
		dir := filepath.Join(os.Getenv("WINDIR"), "Fonts")
		load := func(key string, files ...string) {
			for _, f := range files {
				if data, err := os.ReadFile(filepath.Join(dir, f)); err == nil {
					fontCache[key] = fyne.NewStaticResource(f, data)
					return
				}
			}
		}
		load("regular", "segoeui.ttf")
		load("bold", "seguisb.ttf", "segoeuib.ttf")
		load("italic", "segoeuii.ttf")
		load("mono", "CascadiaMono.ttf", "consola.ttf")
	})
	switch {
	case style.Monospace:
		return fontCache["mono"]
	case style.Bold:
		return fontCache["bold"]
	case style.Italic:
		return fontCache["italic"]
	case style.Symbol:
		return nil
	}
	return fontCache["regular"]
}
