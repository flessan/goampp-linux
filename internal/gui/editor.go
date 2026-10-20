//go:build windows

package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"goampp/internal/config"
	"goampp/internal/core"
	"goampp/internal/vhost"
)

var edUI struct {
	files    *widget.Select
	paths    []string
	labels   []string
	path     string
	pathText *canvas.Text
	content  *widget.Entry
	dirty    *statusPill
	loading  bool
}

func buildEditorPage() (fyne.CanvasObject, fyne.CanvasObject) {
	edUI.path = ""
	edUI.files = widget.NewSelect(nil, func(label string) {
		for i, l := range edUI.labels {
			if l == label && edUI.paths[i] != edUI.path {
				loadFileIntoEditor(edUI.paths[i])
			}
		}
	})
	edUI.files.PlaceHolder = "Choose a file…"

	edUI.content = widget.NewMultiLineEntry()
	edUI.content.TextStyle = fyne.TextStyle{Monospace: true}
	edUI.content.Wrapping = fyne.TextWrapOff
	edUI.content.OnChanged = func(string) {
		if !edUI.loading && edUI.path != "" {
			edUI.dirty.set("Unsaved", colWarning)
		}
	}
	edUI.pathText = mutedText("No file loaded", 12)
	edUI.dirty = newStatusPill()

	reload := lowButton("Reload", theme.ViewRefreshIcon(), func() {
		if edUI.path != "" {
			loadFileIntoEditor(edUI.path)
		}
	})
	open := lowButton("", theme.FolderOpenIcon(), func() {
		if edUI.path != "" {
			core.OpenFolder(filepath.Dir(edUI.path))
		}
	})
	save := importantButton("Save", theme.DocumentSaveIcon(), widget.HighImportance, saveEditorBuffer)
	actions := container.NewHBox(minSized(edUI.files, 240, 0), reload, save)

	top := container.NewBorder(nil, nil, nil, container.NewHBox(container.NewCenter(edUI.dirty.obj), open), edUI.pathText)
	body := newCard(container.NewBorder(container.NewVBox(top, vSpace(4)), nil, nil, nil, edUI.content))
	return actions, body
}

func populateEditorDropdown() {
	if edUI.files == nil {
		return
	}
	edUI.paths, edUI.labels = nil, nil
	add := func(label, path string) {
		if path == "" {
			return
		}
		if _, err := os.Stat(path); err != nil {
			return
		}
		edUI.labels = append(edUI.labels, label)
		edUI.paths = append(edUI.paths, path)
	}
	for _, ms := range core.App.Services {
		if ms.Conf.ConfigFile != "" {
			add(ms.Conf.Name, config.ExpandPath(ms.Conf.ConfigFile, core.App.BaseDir))
		}
	}
	add("GoAMPP config.json", filepath.Join(core.App.BaseDir, "config.json"))
	add("Apache vhosts", config.ExpandPath(core.App.Cfg.Settings.ApacheVhostsInclude, core.App.BaseDir))
	add("Windows hosts", vhost.DefaultHostsFile())
	edUI.files.SetOptions(edUI.labels)

	if edUI.path == "" && len(edUI.paths) > 0 {
		loadFileIntoEditor(edUI.paths[0])
	}
}

func loadFileIntoEditor(path string) {
	if edUI.content == nil {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		core.App.AppendLog(fmt.Sprintf("editor: %v", err))
		dialog.ShowError(err, core.App.Win)
		return
	}
	edUI.loading = true
	edUI.content.SetText(strings.ReplaceAll(string(data), "\r\n", "\n"))
	edUI.loading = false
	edUI.path = path
	edUI.pathText.Text = path
	edUI.pathText.Refresh()
	edUI.dirty.set("Saved", colSuccess)

	found := false
	for i, p := range edUI.paths {
		if p == path {
			edUI.files.SetSelected(edUI.labels[i])
			found = true
		}
	}
	if !found {
		edUI.files.ClearSelected()
	}
	core.App.AppendLog(fmt.Sprintf("editor: loaded %s", path))
}

func saveEditorBuffer() {
	if edUI.path == "" {
		core.App.AppendLog("editor: no file loaded")
		return
	}
	text := edUI.content.Text
	// Keep Windows line endings for files that are read by Windows tools.
	data := []byte(strings.ReplaceAll(text, "\n", "\r\n"))
	if err := vhost.AtomicWrite(edUI.path, data); err != nil {
		core.App.AppendLog(fmt.Sprintf("editor save: %v", err))
		dialog.ShowError(err, core.App.Win)
		return
	}
	edUI.dirty.set("Saved", colSuccess)
	core.App.AppendLog(fmt.Sprintf("editor: saved %s (%d bytes)", edUI.path, len(data)))
	core.SetStatus("Saved " + filepath.Base(edUI.path))
}
