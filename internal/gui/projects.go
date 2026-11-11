//go:build windows

package gui

import (
	"fmt"
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
	"goampp/internal/projects"
	"goampp/internal/uikit"
)

var projUI struct {
	list  *widget.List
	empty fyne.CanvasObject
}

func buildProjectsPage() (fyne.CanvasObject, fyne.CanvasObject) {
	names := make([]string, len(projects.Frameworks))
	for i, f := range projects.Frameworks {
		names[i] = f.Name
	}
	framework := widget.NewSelect(names, nil)
	if len(names) > 0 {
		framework.SetSelected(names[0])
	}
	name := widget.NewEntry()
	name.SetPlaceHolder("my-app")
	domain := widget.NewEntry()
	domain.SetPlaceHolder("defaults to the name")
	ext := widget.NewSelect(core.DomainExtensions, nil)
	ext.SetSelected(core.DomainExtensions[0])

	preview := mutedText("", 12)
	updatePreview := func() {
		n := core.Slugify(name.Text)
		d := strings.TrimSpace(domain.Text)
		if d == "" {
			d = n
		}
		if n == "" {
			preview.Text = "Pick a framework and a name"
		} else {
			preview.Text = fmt.Sprintf("www/%s   →   http://%s%s", n, d, ext.Selected)
		}
		preview.Refresh()
	}
	name.OnChanged = func(string) { updatePreview() }
	domain.OnChanged = func(string) { updatePreview() }
	ext.OnChanged = func(string) { updatePreview() }
	updatePreview()

	create := importantButton("Create project", theme.ContentAddIcon(), widget.HighImportance, nil)
	creator := &uikit.ProjectCreator{
		Prepare: func(request uikit.ProjectRequest) (uikit.ProjectRequest, error) {
			if projects.FrameworkByName(request.Framework) == nil {
				return request, fmt.Errorf("Pick a framework first.")
			}
			request.Name = core.Slugify(request.Name)
			request.Domain = strings.TrimSpace(request.Domain)
			if request.Domain == "" {
				request.Domain = request.Name
			}
			request.Domain += ext.Selected
			return request, nil
		},
		Create: func(request uikit.ProjectRequest) (string, error) {
			err := projects.CreateProject(core.App.Cfg, projects.FrameworkByName(request.Framework), request.Name, request.Domain, core.App.AppendLog)
			return "", err
		},
		Dispatch: core.UiDo,
		Window:   core.App.Win,
		Started: func(request uikit.ProjectRequest) {
			core.App.AppendLog(fmt.Sprintf("projects: creating '%s' (%s) at %s ...", request.Name, request.Framework, request.Domain))
			core.SetStatus(fmt.Sprintf("Creating %s...", request.Name))
			create.Disable()
		},
		Finished: func(request uikit.ProjectRequest, _ string, err error) {
			create.Enable()
			if err != nil {
				core.App.AppendLog("projects: " + err.Error())
				dialog.ShowError(err, core.App.Win)
				return
			}
			core.SetStatus(fmt.Sprintf("Created %s — http://%s", request.Name, request.Domain))
			go core.RestartWebServerIfRunning()
			name.SetText("")
			domain.SetText("")
			refreshProjectList()
			refreshVhostList()
		},
	}
	create.OnTapped = func() {
		creator.Submit(uikit.ProjectRequest{Name: name.Text, Framework: framework.Selected, Domain: domain.Text})
	}

	form := uikit.ProjectForm(framework, name, container.NewBorder(nil, nil, nil, minSized(ext, 110, 0), domain))
	chips := container.NewHBox()
	for _, rs := range core.RuntimeStatuses() {
		p := newStatusPill()
		if rs.OK {
			p.set(rs.Label, colSuccess)
		} else {
			p.set(rs.Label, themeColor(colorMuted))
		}
		chips.Add(p.obj)
	}
	creation := container.NewVBox(form, preview, vSpace(4), container.NewBorder(nil, nil, chips, create))

	projUI.list = widget.NewList(
		func() int { return len(core.App.Cfg.Projects) },
		func() fyne.CanvasObject { return newProjectRow() },
		func(id widget.ListItemID, o fyne.CanvasObject) {
			if id >= len(core.App.Cfg.Projects) {
				return
			}
			o.(*projectRow).bind(id, core.App.Cfg.Projects[id])
		},
	)
	projUI.empty = uikit.EmptyProjects()
	return nil, uikit.ProjectsPage(creation, container.NewStack(projUI.list, projUI.empty),
		"Scaffolds into www/, adds a vhost and the hosts entry.")
}

type projectRow struct {
	widget.BaseWidget
	title, sub        *canvas.Text
	fw                *statusPill
	open, folder, del *widget.Button
}

func newProjectRow() *projectRow {
	r := &projectRow{
		title:  boldText("", 14),
		sub:    mutedText("", 12),
		fw:     newStatusPill(),
		open:   lowButton("", theme.ViewFullScreenIcon(), nil),
		folder: lowButton("", theme.FolderOpenIcon(), nil),
		del:    lowButton("", theme.DeleteIcon(), nil),
	}
	r.ExtendBaseWidget(r)
	return r
}

func (r *projectRow) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(uikit.ProjectRow(r.title, r.sub,
		container.NewHBox(container.NewCenter(r.fw.obj), r.open, r.folder, r.del)))
}

func (r *projectRow) bind(idx int, p config.Project) {
	r.title.Text = p.Name
	r.title.Refresh()
	r.sub.Text = "http://" + p.Domain + "   ·   " + p.DocRoot
	r.sub.Refresh()
	r.fw.set(p.Framework, themeColor(theme.ColorNamePrimary))
	r.open.OnTapped = func() { core.OpenPath("http://" + p.Domain) }
	r.folder.OnTapped = func() { core.OpenFolder(filepath.Join(core.App.BaseDir, "www", p.Name)) }
	r.del.OnTapped = func() {
		dialog.ShowConfirm("Delete project",
			fmt.Sprintf("Delete %s?\n\nThe folder www/%s and its virtual host are removed.", p.Name, p.Name),
			func(ok bool) {
				if !ok {
					return
				}
				core.DeleteProject(idx)
				refreshProjectList()
				refreshVhostList()
			}, core.App.Win)
	}
}

func refreshProjectList() {
	if projUI.list == nil {
		return
	}
	projUI.list.Refresh()
	if len(core.App.Cfg.Projects) == 0 {
		projUI.empty.Show()
	} else {
		projUI.empty.Hide()
	}
}
