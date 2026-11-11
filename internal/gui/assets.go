//go:build windows

package gui

import (
	"fyne.io/fyne/v2"
	"goampp/internal/core"
	"goampp/internal/uikit"
	"path"
)

func logoResource() fyne.Resource { return uikit.Asset("logo.png") }

// serviceIcon preserves the Windows preference for logos over ICO assets.
func serviceIcon(name string) fyne.Resource {
	if file, ok := core.ServiceIconPNGs[name]; ok {
		if res := uikit.Asset(path.Join("logos", file)); res != nil {
			return res
		}
	}
	if file, ok := core.ServiceIconFiles[name]; ok {
		return uikit.Asset(path.Join("icons", file))
	}
	return nil
}
