//go:build windows

package gui

import (
	"fmt"
	"net/url"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"goampp/internal/config"
	"goampp/internal/core"
	"goampp/internal/pathenv"
	"goampp/internal/vhost"
)

func buildSettingsPage() (fyne.CanvasObject, fyne.CanvasObject) {
	// Appearance
	themes := []string{themeSystem, themeLight, themeDark}
	themeSeg := newSegmented(themes, func(i int) {
		core.App.Fyne.Preferences().SetString(themePrefKey, themes[i])
		core.App.Fyne.Settings().SetTheme(newGoamppTheme(themes[i]))
	})
	current := core.App.Fyne.Preferences().StringWithFallback(themePrefKey, themeSystem)
	for i, t := range themes {
		if t == current {
			themeSeg.setSelected(i, false)
		}
	}
	appearance := settingsRow("Theme", "System follows the Windows light/dark setting.", themeSeg.obj)

	// Paths
	paths := container.NewVBox()
	addPath := func(label, path string, isDir bool) {
		open := lowButton("", theme.FolderOpenIcon(), func() {
			if isDir {
				core.OpenFolder(path)
			} else {
				core.OpenFolder(filepath.Dir(path))
			}
		})
		if len(paths.Objects) > 0 {
			paths.Add(rowDivider())
		}
		value := truncLabel(path)
		value.Importance = widget.LowImportance
		paths.Add(container.New(layout.NewCustomPaddedLayout(0, 0, 10, 4),
			container.NewBorder(nil, nil, minSized(widget.NewLabel(label), 120, 0), open, value)))
	}
	addPath("Install folder", core.App.BaseDir, true)
	addPath("Config", filepath.Join(core.App.BaseDir, "config.json"), false)
	addPath("Web root", filepath.Join(core.App.BaseDir, "www"), true)
	addPath("Apache vhosts", config.ExpandPath(core.App.Cfg.Settings.ApacheVhostsInclude, core.App.BaseDir), false)
	addPath("Nginx sites", config.ExpandPath(core.App.Cfg.Settings.NginxSitesDir, core.App.BaseDir), true)
	addPath("Hosts file", vhost.DefaultHostsFile(), false)
	addPath("Download cache", filepath.Join(core.App.BaseDir, "downloads"), true)

	// Tools
	tools := container.NewHBox(
		lowButton("Edit config.json", theme.DocumentCreateIcon(), func() {
			showPage("editor")
			loadFileIntoEditor(filepath.Join(core.App.BaseDir, "config.json"))
		}),
		lowButton("Reload config", theme.ViewRefreshIcon(), func() {
			if err := core.ReloadConfig(); err != nil {
				core.App.AppendLog("reload: " + err.Error())
				return
			}
			core.App.AppendLog("config reloaded")
			rebuildContent()
		}),
		lowButton("Add to PATH", theme.ContentAddIcon(), func() {
			n, err := pathenv.AddGoamppToUserPath()
			switch {
			case err != nil:
				core.App.AppendLog("path: " + err.Error())
			case n == 0:
				core.App.AppendLog("path: already on PATH (no changes)")
			default:
				core.App.AppendLog(fmt.Sprintf("path: added %d goampp bin dirs to user PATH — open a new terminal to use them", n))
			}
		}),
		lowButton("Remove from PATH", theme.ContentRemoveIcon(), func() {
			n, err := pathenv.RemoveGoamppFromUserPath()
			switch {
			case err != nil:
				core.App.AppendLog("path: " + err.Error())
			case n == 0:
				core.App.AppendLog("path: nothing to remove")
			default:
				core.App.AppendLog(fmt.Sprintf("path: removed %d goampp entries from user PATH", n))
			}
		}),
		lowButton("psql console", theme.ComputerIcon(), core.OpenPsqlConsole),
		layout.NewSpacer(),
		lowButton("Quit", theme.NewErrorThemedResource(theme.LogoutIcon()), quitApp),
	)

	// About
	var logo fyne.CanvasObject = letterAvatar("GoAMPP", 32)
	if res := logoResource(); res != nil {
		img := canvas.NewImageFromResource(res)
		img.FillMode = canvas.ImageFillContain
		logo = container.NewGridWrap(fyne.NewSize(32, 32), img)
	}
	gh, _ := url.Parse("https://github.com/imtaqin/goampp")
	site, _ := url.Parse("https://imtaqin.id")
	donate, _ := url.Parse("https://saweria.co/fdciabdul")
	about := container.NewBorder(nil, nil, container.NewCenter(logo),
		container.NewHBox(widget.NewHyperlink("GitHub", gh), widget.NewHyperlink("imtaqin.id", site), widget.NewHyperlink("Donate", donate)),
		container.NewVBox(boldText("GoAMPP "+core.Version, 13), mutedText("Native Windows local dev stack · by imtaqin", 11.5)))

	content := container.NewVBox(
		sectionLabel("APPEARANCE"), newListCard(appearance),
		sectionLabel("PATHS"), newListCard(paths),
		sectionLabel("TOOLS"), newListCard(container.New(layout.NewCustomPaddedLayout(4, 4, 6, 6), tools)),
		sectionLabel("ABOUT"), newListCard(container.New(layout.NewCustomPaddedLayout(8, 8, 12, 8), about)),
	)
	return nil, container.NewVScroll(content)
}

func sectionLabel(s string) fyne.CanvasObject {
	return container.New(layout.NewCustomPaddedLayout(8, 2, 2, 0), mutedText(s, 10.5))
}

// settingsRow is a label + description on the left with a control on the right.
func settingsRow(title, desc string, control fyne.CanvasObject) fyne.CanvasObject {
	left := container.NewVBox(boldText(title, 12.5), mutedText(desc, 11.5))
	return container.New(layout.NewCustomPaddedLayout(8, 8, 12, 8),
		container.NewBorder(nil, nil, left, container.NewCenter(control)))
}
