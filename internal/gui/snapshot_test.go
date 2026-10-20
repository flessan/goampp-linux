//go:build windows

package gui

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"goampp/internal/config"
	"goampp/internal/core"
	_ "goampp/internal/services"
)

// TestGUISnapshots renders every page off-screen to PNG files so the layout
// can be reviewed without a desktop session. It only runs when
// GOAMPP_SNAPSHOT names an output directory.
func TestGUISnapshots(t *testing.T) {
	out := os.Getenv("GOAMPP_SNAPSHOT")
	if out == "" {
		t.Skip("set GOAMPP_SNAPSHOT=<dir> to render GUI snapshots")
	}
	base := t.TempDir()
	cfg, err := config.LoadConfig(base)
	if err != nil {
		t.Fatal(err)
	}
	core.App = &core.State{BaseDir: base, Cfg: cfg}
	core.App.Services = core.BuildManagedServices(cfg, nil)
	for i := 0; i < 6; i++ {
		core.App.AppendLog("example log line")
	}
	core.App.AppendLog("GoAMPP started — 30 services, 1 vhost loaded from config.json")

	for _, variant := range []string{themeDark, themeLight} {
		a := test.NewTempApp(t)
		a.Settings().SetTheme(newGoamppTheme(variant))
		w := test.NewTempWindow(t, nil)
		core.App.Fyne, core.App.Win = a, w
		gui.current = ""
		rebuildContent()
		w.Resize(fyne.NewSize(1120, 720))
		for _, p := range gui.pages {
			showPage(p.key)
			img := w.Canvas().Capture()
			f, err := os.Create(filepath.Join(out, variant+"_"+p.key+".png"))
			if err != nil {
				t.Fatal(err)
			}
			_ = png.Encode(f, img)
			f.Close()
		}
	}
}
