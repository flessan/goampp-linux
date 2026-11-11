//go:build linux

package linuxapp

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// TestLinuxPageSnapshots captures actual Linux page builders with the Fyne test
// driver. It intentionally does not touch system service or hosts configuration.
func TestLinuxPageSnapshots(t *testing.T) {
	out := os.Getenv("GOAMPP_SNAPSHOT")
	if out == "" {
		t.Skip("set GOAMPP_SNAPSHOT to capture snapshots")
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	projectDir := t.TempDir()
	for _, variant := range []string{"Light", "Dark"} {
		a := test.NewTempApp(t)
		a.Settings().SetTheme(newLinuxGoamppTheme(variant))
		w := test.NewTempWindow(t, nil)
		editorActions, editor := buildLinuxConfigEditorPage(w)
		vhostActions, hosts := buildLinuxVHostsPage(w, projectDir)
		pages := []*linuxPage{
			{key: "services", title: "Services", icon: theme.ComputerIcon(), body: widget.NewLabel("")},
			{key: "projects", title: "Projects", icon: theme.FolderIcon(), body: widget.NewLabel("")},
			{key: "vhosts", title: "Virtual Hosts", icon: theme.HomeIcon(), actions: vhostActions, body: hosts},
			{key: "editor", title: "Config Editor", icon: theme.DocumentCreateIcon(), actions: editorActions, body: editor},
			{key: "startup", title: "Startup", icon: theme.MediaPlayIcon(), body: widget.NewLabel("")},
			{key: "settings", title: "Settings", icon: theme.SettingsIcon(), body: widget.NewLabel("")},
		}
		attachLinuxShell(a, w, pages, "editor")
		w.Resize(fyne.NewSize(1080, 680))
		for _, key := range []string{"editor", "vhosts"} {
			linuxShowPage(key)
			file, err := os.Create(filepath.Join(out, variant+"_"+key+".png"))
			if err != nil {
				t.Fatal(err)
			}
			if err := png.Encode(file, w.Canvas().Capture()); err != nil {
				file.Close()
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
		}
	}
}
