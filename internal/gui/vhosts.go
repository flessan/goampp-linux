//go:build windows

package gui

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"goampp/internal/config"
	"goampp/internal/core"
)

var vhostUI struct {
	list     *widget.List
	empty    fyne.CanvasObject
	selected int

	name, docroot, port *widget.Entry
	ext                 *widget.Select
	server              *widget.Select
	formTitle           *canvas.Text
	toggleBtn, delBtn   *widget.Button
}

func buildVhostsPage() (fyne.CanvasObject, fyne.CanvasObject) {
	vhostUI.selected = -1

	vhostUI.list = widget.NewList(
		func() int { return len(core.App.Cfg.Vhosts) },
		func() fyne.CanvasObject { return newVhostRow() },
		func(id widget.ListItemID, o fyne.CanvasObject) {
			if id < len(core.App.Cfg.Vhosts) {
				o.(*vhostRow).bind(core.App.Cfg.Vhosts[id])
			}
		},
	)
	vhostUI.list.OnSelected = func(id widget.ListItemID) { selectVhost(id) }
	vhostUI.empty = container.NewCenter(mutedText("No virtual hosts yet — add one on the right.", 13))
	left := newTitledCard("Domains", "Select a domain to edit it.", nil, container.NewStack(vhostUI.list, vhostUI.empty))

	vhostUI.name = widget.NewEntry()
	vhostUI.name.SetPlaceHolder("myapp")
	vhostUI.ext = widget.NewSelect(core.DomainExtensions, nil)
	vhostUI.ext.SetSelected(core.DomainExtensions[0])
	vhostUI.port = widget.NewEntry()
	vhostUI.port.SetText("80")
	vhostUI.server = widget.NewSelect([]string{"apache", "nginx", "both"}, nil)
	vhostUI.server.SetSelected("apache")
	vhostUI.docroot = widget.NewEntry()
	vhostUI.docroot.SetPlaceHolder(`{base}/www/myapp/public`)
	browse := lowButton("", theme.FolderOpenIcon(), func() {
		d := dialog.NewFolderOpen(func(u fyne.ListableURI, err error) {
			if err == nil && u != nil {
				vhostUI.docroot.SetText(u.Path())
			}
		}, core.App.Win)
		d.Show()
	})

	form := container.New(layout.NewFormLayout(),
		widget.NewLabel("Domain"), container.NewBorder(nil, nil, nil, minSized(vhostUI.ext, 110, 0), vhostUI.name),
		widget.NewLabel("Document root"), container.NewBorder(nil, nil, nil, browse, vhostUI.docroot),
		widget.NewLabel("Port"), container.NewBorder(nil, nil, minSized(vhostUI.port, 90, 0), nil),
		widget.NewLabel("Server"), minSized(vhostUI.server, 140, 0),
	)

	save := importantButton("Save", theme.DocumentSaveIcon(), widget.HighImportance, saveVhostForm)
	vhostUI.toggleBtn = lowButton("Disable", theme.VisibilityOffIcon(), toggleSelectedVhost)
	vhostUI.delBtn = importantButton("Delete", theme.DeleteIcon(), widget.DangerImportance, deleteSelectedVhost)
	vhostUI.formTitle = boldText("New virtual host", 15)
	right := newCard(container.NewVBox(
		vhostUI.formTitle,
		mutedText("Saved changes are applied to hosts and the web server automatically.", 11.5),
		vSpace(4), form, vSpace(6),
		container.NewHBox(save, vhostUI.toggleBtn, layout.NewSpacer(), vhostUI.delBtn),
	))

	newBtn := importantButton("New", theme.ContentAddIcon(), widget.MediumImportance, func() {
		vhostUI.list.UnselectAll()
		clearVhostForm()
	})
	applyBtn := lowButton("Re-apply all", theme.ViewRefreshIcon(), core.ApplyVhostsAuto)
	actions := container.NewHBox(applyBtn, newBtn)

	split := container.NewHSplit(left, container.NewVBox(right))
	split.Offset = 0.52
	clearVhostForm()
	return actions, split
}

type vhostRow struct {
	widget.BaseWidget
	state        *statusPill
	domain, info *canvas.Text
}

func newVhostRow() *vhostRow {
	r := &vhostRow{state: newStatusPill(), domain: boldText("", 14), info: mutedText("", 12)}
	r.ExtendBaseWidget(r)
	return r
}

func (r *vhostRow) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewBorder(nil, nil,
		container.NewHBox(hSpace(4), container.NewVBox(r.domain, r.info)), container.NewCenter(r.state.obj)))
}

func (r *vhostRow) bind(v config.Vhost) {
	r.domain.Text = v.Domain
	r.domain.Refresh()
	port := v.Port
	if port == 0 {
		port = 80
	}
	server := v.ServerType
	if server == "" {
		server = "apache"
	}
	r.info.Text = fmt.Sprintf("%s  ·  :%d  ·  %s", core.TruncateMid(config.ExpandPath(v.DocRoot, core.App.BaseDir), 48), port, server)
	r.info.Refresh()
	if v.Enabled {
		r.state.set("Enabled", colSuccess)
	} else {
		r.state.set("Disabled", themeColor(colorMuted))
	}
}

func selectVhost(id int) {
	if id < 0 || id >= len(core.App.Cfg.Vhosts) {
		return
	}
	vhostUI.selected = id
	v := core.App.Cfg.Vhosts[id]
	name, ext := core.SplitDomain(v.Domain)
	vhostUI.name.SetText(name)
	vhostUI.ext.SetSelected(ext)
	vhostUI.docroot.SetText(v.DocRoot)
	port := v.Port
	if port == 0 {
		port = 80
	}
	vhostUI.port.SetText(strconv.Itoa(port))
	server := v.ServerType
	if server == "" {
		server = "apache"
	}
	vhostUI.server.SetSelected(server)
	vhostUI.formTitle.Text = "Edit " + v.Domain
	vhostUI.formTitle.Refresh()
	vhostUI.delBtn.Enable()
	vhostUI.toggleBtn.Enable()
	if v.Enabled {
		vhostUI.toggleBtn.SetText("Disable")
		vhostUI.toggleBtn.SetIcon(theme.VisibilityOffIcon())
	} else {
		vhostUI.toggleBtn.SetText("Enable")
		vhostUI.toggleBtn.SetIcon(theme.VisibilityIcon())
	}
}

func clearVhostForm() {
	vhostUI.selected = -1
	vhostUI.name.SetText("")
	vhostUI.ext.SetSelected(core.DomainExtensions[0])
	vhostUI.docroot.SetText("")
	vhostUI.port.SetText("80")
	vhostUI.server.SetSelected(activeServerType())
	vhostUI.formTitle.Text = "New virtual host"
	vhostUI.formTitle.Refresh()
	vhostUI.delBtn.Disable()
	vhostUI.toggleBtn.Disable()
}

func activeServerType() string {
	if core.ActiveWebServer() == "Nginx" {
		return "nginx"
	}
	return "apache"
}

func saveVhostForm() {
	v, err := core.ParseVhostInput(vhostUI.name.Text, vhostUI.ext.Selected, vhostUI.docroot.Text,
		vhostUI.port.Text, vhostUI.server.Selected)
	if err != nil {
		dialog.ShowError(err, core.App.Win)
		return
	}
	idx := vhostUI.selected
	if idx >= 0 && idx < len(core.App.Cfg.Vhosts) {
		v.Enabled = core.App.Cfg.Vhosts[idx].Enabled
		core.App.Cfg.Vhosts[idx] = v
	} else {
		v.Enabled = true
		core.App.Cfg.Vhosts = append(core.App.Cfg.Vhosts, v)
		idx = len(core.App.Cfg.Vhosts) - 1
	}
	if err := config.SaveConfig(core.App.BaseDir, core.App.Cfg); err != nil {
		dialog.ShowError(err, core.App.Win)
		return
	}
	core.App.AppendLog("vhost: saved " + v.Domain)
	refreshVhostList()
	vhostUI.list.Select(idx)
	core.ApplyVhostsAuto()
}

func toggleSelectedVhost() {
	idx := vhostUI.selected
	if idx < 0 || idx >= len(core.App.Cfg.Vhosts) {
		return
	}
	v := &core.App.Cfg.Vhosts[idx]
	v.Enabled = !v.Enabled
	if err := config.SaveConfig(core.App.BaseDir, core.App.Cfg); err != nil {
		v.Enabled = !v.Enabled
		dialog.ShowError(err, core.App.Win)
		return
	}
	state := "disabled"
	if v.Enabled {
		state = "enabled"
	}
	core.App.AppendLog(fmt.Sprintf("vhost: %s %s", v.Domain, state))
	refreshVhostList()
	selectVhost(idx)
	core.ApplyVhostsAuto()
}

func deleteSelectedVhost() {
	idx := vhostUI.selected
	if idx < 0 || idx >= len(core.App.Cfg.Vhosts) {
		return
	}
	domain := core.App.Cfg.Vhosts[idx].Domain
	dialog.ShowConfirm("Delete virtual host", fmt.Sprintf("Remove %s? Files in its document root are kept.", domain), func(ok bool) {
		if !ok {
			return
		}
		core.App.Cfg.Vhosts = append(core.App.Cfg.Vhosts[:idx], core.App.Cfg.Vhosts[idx+1:]...)
		if err := config.SaveConfig(core.App.BaseDir, core.App.Cfg); err != nil {
			dialog.ShowError(err, core.App.Win)
			return
		}
		core.App.AppendLog("vhost: deleted " + domain)
		vhostUI.list.UnselectAll()
		clearVhostForm()
		refreshVhostList()
		core.ApplyVhostsAuto()
	}, core.App.Win)
}

func refreshVhostList() {
	if vhostUI.list == nil {
		return
	}
	vhostUI.list.Refresh()
	if len(core.App.Cfg.Vhosts) == 0 {
		vhostUI.empty.Show()
	} else {
		vhostUI.empty.Hide()
	}
}
