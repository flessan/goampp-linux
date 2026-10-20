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
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"goampp/internal/config"
	"goampp/internal/core"
	"goampp/internal/packages"
)

// svcCard is one service row on the Services page.
type svcCard struct {
	ms  *core.ManagedService
	obj fyne.CanvasObject

	version   *canvas.Text
	badge     *canvas.Text
	status    *statusPill
	statusBox fyne.CanvasObject
	primary   *widget.Button
	verSel    *widget.Select
	more      *widget.Button
	progress  *widget.ProgressBar

	installing bool
}

type svcSection struct {
	group int32
	rows  *fyne.Container
	obj   fyne.CanvasObject
}

var svcUI struct {
	cards    []*svcCard
	byName   map[string]*svcCard
	sections []*svcSection
	tab      int
	query    string

	webSelect  *widget.Select
	refreshBtn *widget.Button
	updateBtn  *widget.Button
	scroll     *container.Scroll
}

var updateApplyBusy bool

var svcTabs = []struct {
	label  string
	groups []int32
}{
	{"All", nil},
	{"Web", []int32{core.GroupWeb}},
	{"Databases", []int32{core.GroupDatabase}},
	{"Languages", []int32{core.GroupLanguage}},
	{"Tools", []int32{core.GroupTool}},
}

const (
	statusColW  = 120
	versionColW = 78
	actionColW  = 86
)

func buildServicesPage() (fyne.CanvasObject, fyne.CanvasObject) {
	svcUI.cards = nil
	svcUI.byName = map[string]*svcCard{}
	svcUI.sections = nil

	svcUI.webSelect = widget.NewSelect([]string{"Apache", "Nginx"}, nil)
	svcUI.webSelect.SetSelected(core.ActiveWebServer())
	svcUI.webSelect.OnChanged = func(choice string) {
		core.SetActiveWebServer(choice)
		refreshServiceList()
	}
	svcUI.refreshBtn = lowButton("Check updates", theme.ViewRefreshIcon(), func() {
		if !packages.UpdateOperationMu.TryLock() {
			return
		}
		packages.UpdateOperationMu.Unlock()
		setUpdateRefreshBusy(true)
		go core.RefreshAllServiceUpdates()
	})
	svcUI.updateBtn = importantButton("Up to date", theme.DownloadIcon(), widget.HighImportance, showAvailableUpdatesMenu)
	svcUI.updateBtn.Disable()
	actions := container.NewHBox(
		container.NewCenter(mutedText("Web server", 11.5)), minSized(svcUI.webSelect, 96, 0),
		svcUI.refreshBtn, svcUI.updateBtn,
	)

	labels := make([]string, len(svcTabs))
	for i, t := range svcTabs {
		labels[i] = t.label
	}
	tabs := newSegmented(labels, func(i int) {
		svcUI.tab = i
		applyServiceFilter()
	})
	tabs.setSelected(svcUI.tab, false)
	search := widget.NewEntry()
	search.SetPlaceHolder("Search")
	search.ActionItem = widget.NewIcon(theme.NewDisabledResource(theme.SearchIcon()))
	search.SetText(svcUI.query)
	search.OnChanged = func(s string) {
		svcUI.query = strings.ToLower(strings.TrimSpace(s))
		applyServiceFilter()
	}
	restart := lowButton("Restart stack", theme.MediaReplayIcon(), func() { go core.RestartStack() })
	toolbar := container.NewBorder(nil, nil, tabs.obj, container.NewHBox(restart, minSized(search, 190, 0)))

	order := []struct {
		group int32
		title string
	}{
		{core.GroupWeb, "WEB SERVERS"},
		{core.GroupDatabase, "DATABASES & CACHES"},
		{core.GroupLanguage, "LANGUAGES & RUNTIMES"},
		{core.GroupTool, "TOOLS"},
	}
	list := container.NewVBox()
	for _, o := range order {
		for _, ms := range core.App.Services {
			if core.ServiceGroup(ms) != o.group {
				continue
			}
			c := newServiceCard(ms)
			svcUI.cards = append(svcUI.cards, c)
			svcUI.byName[ms.Conf.Name] = c
		}
		rows := container.NewVBox()
		sec := &svcSection{group: o.group, rows: rows}
		sec.obj = container.NewVBox(
			container.New(layout.NewCustomPaddedLayout(8, 2, 2, 0), mutedText(o.title, 10.5)),
			newListCard(rows),
		)
		svcUI.sections = append(svcUI.sections, sec)
		list.Add(sec.obj)
	}

	svcUI.scroll = container.NewVScroll(list)
	applyServiceFilter()
	body := container.NewBorder(container.NewVBox(toolbar, vSpace(2)), nil, nil, nil, svcUI.scroll)
	return actions, body
}

// newListCard is a bordered surface with no inner padding, for row lists.
func newListCard(content fyne.CanvasObject) fyne.CanvasObject {
	bg := canvas.NewRectangle(themeColor(colorCard))
	bg.CornerRadius = theme.Size(theme.SizeNameCardRadius)
	bg.StrokeColor = themeColor(colorCardBorder)
	bg.StrokeWidth = 1
	return container.NewStack(bg, container.New(layout.NewCustomPaddedLayout(2, 2, 0, 0), content))
}

func newServiceCard(ms *core.ManagedService) *svcCard {
	c := &svcCard{ms: ms}
	name := ms.Conf.Name

	c.version = mutedText("", 11.5)
	c.badge = canvas.NewText("", themeColor(theme.ColorNamePrimary))
	c.badge.TextSize = 11
	c.badge.TextStyle = fyne.TextStyle{Bold: true}

	title := boldText(name, 12.5)
	left := container.NewHBox(
		hSpace(4),
		container.NewCenter(serviceAvatar(name, 20)),
		hSpace(2),
		container.NewCenter(title),
		container.NewCenter(c.version),
		container.NewCenter(c.badge),
	)

	c.status = newStatusPill()
	c.progress = widget.NewProgressBar()
	c.progress.TextFormatter = func() string { return "" }
	c.progress.Hide()
	c.statusBox = container.NewStack(
		container.NewHBox(c.status.obj),
		container.NewCenter(minSized(c.progress, statusColW-12, 0)),
	)

	var verCol fyne.CanvasObject = hSpace(versionColW)
	if spec, ok := packages.DownloadCatalog[name]; ok && len(spec.Variants) > 0 {
		opts := make([]string, len(spec.Variants))
		for i, v := range spec.Variants {
			opts[i] = v.Version
		}
		c.verSel = widget.NewSelect(opts, nil)
		c.verSel.PlaceHolder = "Ver"
		verCol = minSized(c.verSel, versionColW, 0)
	}

	c.primary = lowButton("", nil, nil)
	c.primary.Alignment = widget.ButtonAlignLeading
	c.more = lowButton("", theme.MoreHorizontalIcon(), nil)
	c.more.OnTapped = func() { showMenuBelow(serviceMenu(c), c.more) }

	right := container.NewHBox(
		minSized(c.statusBox, statusColW, 0),
		verCol,
		minSized(c.primary, actionColW, 0),
		c.more,
	)
	row := container.New(layout.NewCustomPaddedLayout(3, 3, 6, 6), container.NewBorder(nil, nil, left, right))
	c.obj = newContextSurface(row, func(pos fyne.Position) {
		widget.ShowPopUpMenuAtPosition(serviceMenu(c), core.App.Win.Canvas(), pos)
	})
	c.refresh()
	return c
}

func (c *svcCard) isWebInactive() bool {
	return core.IsWebServerName(c.ms.Conf.Name) && c.ms.Conf.Name != core.ActiveWebServer()
}

func (c *svcCard) setAction(label string, icon fyne.Resource, fn func()) {
	b := c.primary
	b.SetText(label)
	b.SetIcon(icon)
	b.OnTapped = fn
	if fn == nil {
		b.Disable()
	} else {
		b.Enable()
	}
}

func (c *svcCard) refresh() {
	ms := c.ms
	name := ms.Conf.Name
	_, inCatalog := packages.DownloadCatalog[name]
	installed := !inCatalog || packages.IsInstalled(name, core.App.BaseDir)
	running := ms.Service != nil && ms.Service.Running()
	isRuntime := strings.EqualFold(ms.Conf.Kind, "runtime")

	version, update := core.DisplayedServiceVersion(ms)
	c.version.Text = version
	c.version.Refresh()
	c.badge.Text = ""
	if update && installed {
		c.badge.Text = "update"
	}
	c.badge.Refresh()

	port := ""
	if ms.Conf.Port > 0 {
		port = fmt.Sprintf("  :%d", ms.Conf.Port)
	}
	muted := themeColor(colorMuted)
	accent := themeColor(theme.ColorNamePrimary)
	switch {
	case c.installing:
		c.status.set("Installing", accent)
		c.setAction("Wait", theme.DownloadIcon(), nil)
	case c.isWebInactive():
		c.status.set("Inactive", muted)
		c.setAction("Use", theme.ConfirmIcon(), func() { svcUI.webSelect.SetSelected(name) })
	case !installed:
		c.status.set("Not installed", muted)
		c.setAction("Install", theme.NewPrimaryThemedResource(theme.DownloadIcon()), func() {
			c.installing = true
			c.refresh()
			core.StartService(ms)
		})
	case running:
		c.status.set("Running"+port, colSuccess)
		c.setAction("Stop", theme.NewErrorThemedResource(theme.MediaStopIcon()), func() { core.ToggleService(ms) })
	case ms.Service == nil && isRuntime:
		c.status.set("Installed", accent)
		c.setAction("Shell", theme.ComputerIcon(), func() { core.OpenTerminalForService(name) })
	case ms.Service == nil:
		c.status.set("Installed", accent)
		if ms.Conf.OpenURL != "" {
			c.setAction("Open", theme.ViewFullScreenIcon(), func() { core.StartService(ms) })
		} else {
			c.setAction("Open", theme.ViewFullScreenIcon(), nil)
		}
	default:
		c.status.set("Stopped"+port, muted)
		c.setAction("Start", theme.NewSuccessThemedResource(theme.MediaPlayIcon()), func() { core.ToggleService(ms) })
	}

	if c.verSel != nil {
		c.verSel.OnChanged = nil
		c.verSel.SetSelected(packages.EffectiveVariant(name, ms.Conf.ActiveVersion))
		c.verSel.OnChanged = func(v string) { core.SwitchServiceVersion(ms, v) }
		if c.installing {
			c.verSel.Disable()
		} else {
			c.verSel.Enable()
		}
	}
}

func setCardProgress(name string, active bool, fraction float64) {
	c := svcUI.byName[name]
	if c == nil {
		return
	}
	c.installing = active
	if active {
		c.progress.SetValue(fraction)
		c.progress.Show()
		c.status.obj.Hide()
	} else {
		c.progress.Hide()
		c.status.obj.Show()
	}
	c.refresh()
}

func serviceMenu(c *svcCard) *fyne.Menu {
	ms := c.ms
	name := ms.Conf.Name
	spec, inCatalog := packages.DownloadCatalog[name]
	installed := !inCatalog || packages.IsInstalled(name, core.App.BaseDir)
	running := ms.Service != nil && ms.Service.Running()

	var items []*fyne.MenuItem
	if running {
		items = append(items, fyne.NewMenuItemWithIcon("Restart", theme.MediaReplayIcon(), func() { core.RestartService(ms) }))
	}
	if ms.Conf.OpenURL != "" {
		items = append(items, fyne.NewMenuItemWithIcon("Open in browser", theme.ViewFullScreenIcon(), func() { core.OpenPath(ms.Conf.OpenURL) }))
	}
	if ms.Conf.ConfigFile != "" {
		items = append(items, fyne.NewMenuItemWithIcon("Edit config", theme.DocumentCreateIcon(), func() { openConfigInEditor(ms) }))
	}
	if strings.EqualFold(ms.Conf.Kind, "runtime") || name == "PHP-FPM" {
		items = append(items, fyne.NewMenuItemWithIcon("Open terminal", theme.ComputerIcon(), func() { core.OpenTerminalForService(name) }))
	}
	if inCatalog && installed {
		dir := filepath.Join(core.App.BaseDir, filepath.FromSlash(spec.InstallDir))
		items = append(items, fyne.NewMenuItemWithIcon("Open folder", theme.FolderOpenIcon(), func() { core.OpenFolder(dir) }))
	}
	if inCatalog && len(spec.Variants) > 0 {
		current := packages.EffectiveVariant(name, ms.Conf.ActiveVersion)
		var sub []*fyne.MenuItem
		for _, v := range spec.Variants {
			v := v
			it := fyne.NewMenuItem(v.Version, func() { core.SwitchServiceVersion(ms, v.Version) })
			it.Checked = v.Version == current
			sub = append(sub, it)
		}
		ver := fyne.NewMenuItem("Switch version", nil)
		ver.ChildMenu = fyne.NewMenu("", sub...)
		items = append(items, ver)
	}
	if core.StartupServiceEligible(ms) {
		auto := fyne.NewMenuItem("Start with GoAMPP", func() {
			if err := core.SetServiceAutoStart(name, !core.AutoStartContains(name)); err != nil {
				core.App.AppendLog("startup services: " + err.Error())
			}
		})
		auto.Checked = core.AutoStartContains(name)
		items = append(items, auto)
	}
	if inCatalog && installed {
		items = append(items, fyne.NewMenuItemSeparator(),
			fyne.NewMenuItemWithIcon("Uninstall…", theme.DeleteIcon(), func() { confirmUninstall(ms) }))
	}
	if len(items) == 0 {
		items = append(items, fyne.NewMenuItem("No actions", nil))
		items[0].Disabled = true
	}
	return fyne.NewMenu(name, items...)
}

func confirmUninstall(ms *core.ManagedService) {
	msg := fmt.Sprintf("Remove %s from this machine?\n\nIts install folder is deleted, including any data stored inside it.", ms.Conf.Name)
	d := dialog.NewConfirm("Uninstall "+ms.Conf.Name, msg, func(ok bool) {
		if ok {
			go core.UninstallService(ms)
		}
	}, core.App.Win)
	d.SetConfirmImportance(widget.DangerImportance)
	d.SetConfirmText("Uninstall")
	d.Show()
}

func openConfigInEditor(ms *core.ManagedService) {
	showPage("editor")
	loadFileIntoEditor(config.ExpandPath(ms.Conf.ConfigFile, core.App.BaseDir))
}

func applyServiceFilter() {
	groups := svcTabs[svcUI.tab].groups
	for _, sec := range svcUI.sections {
		wantGroup := len(groups) == 0
		for _, g := range groups {
			if g == sec.group {
				wantGroup = true
			}
		}
		var objs []fyne.CanvasObject
		if wantGroup {
			for _, c := range svcUI.cards {
				if core.ServiceGroup(c.ms) != sec.group {
					continue
				}
				if svcUI.query != "" && !strings.Contains(strings.ToLower(c.ms.Conf.Name), svcUI.query) {
					continue
				}
				if len(objs) > 0 {
					objs = append(objs, rowDivider())
				}
				objs = append(objs, c.obj)
			}
		}
		sec.rows.Objects = objs
		sec.rows.Refresh()
		if len(objs) == 0 {
			sec.obj.Hide()
		} else {
			sec.obj.Show()
		}
	}
	if svcUI.scroll != nil {
		svcUI.scroll.ScrollToTop()
	}
}

func rowDivider() fyne.CanvasObject {
	line := canvas.NewRectangle(themeColor(theme.ColorNameSeparator))
	line.SetMinSize(fyne.NewSize(1, 1))
	return container.New(layout.NewCustomPaddedLayout(0, 0, 10, 10), line)
}

// refreshServiceList re-renders every row from live state. It must run on
// the GUI goroutine; background code goes through uiDo(refreshServiceList).
func refreshServiceList() {
	for _, c := range svcUI.cards {
		c.refresh()
	}
	if svcUI.webSelect != nil && svcUI.webSelect.Selected != core.ActiveWebServer() {
		cb := svcUI.webSelect.OnChanged
		svcUI.webSelect.OnChanged = nil
		svcUI.webSelect.SetSelected(core.ActiveWebServer())
		svcUI.webSelect.OnChanged = cb
	}
	refreshUpdateApplyButton()
	refreshStackSummary()
}

// ---- updates ----

func setUpdateRefreshBusy(busy bool) {
	b := svcUI.refreshBtn
	if b == nil {
		return
	}
	if busy {
		b.SetText("Checking…")
		b.Disable()
		if svcUI.updateBtn != nil {
			svcUI.updateBtn.Disable()
		}
		return
	}
	b.SetText("Check updates")
	if !updateApplyBusy {
		b.Enable()
	}
	refreshUpdateApplyButton()
}

func setUpdateApplyBusy(busy bool) {
	updateApplyBusy = busy
	if svcUI.refreshBtn != nil {
		if busy {
			svcUI.refreshBtn.Disable()
		} else {
			svcUI.refreshBtn.Enable()
		}
	}
	if svcUI.updateBtn == nil {
		return
	}
	if busy {
		svcUI.updateBtn.SetText("Updating…")
		svcUI.updateBtn.Disable()
		return
	}
	refreshUpdateApplyButton()
}

func refreshUpdateApplyButton() {
	b := svcUI.updateBtn
	if b == nil || updateApplyBusy {
		return
	}
	if n := len(core.AvailableServiceUpdates()); n > 0 {
		b.SetText(fmt.Sprintf("Update %d", n))
		b.Importance = widget.HighImportance
		b.Enable()
	} else {
		b.SetText("Up to date")
		b.Importance = widget.LowImportance
		b.Disable()
	}
	b.Refresh()
}

func showAvailableUpdatesMenu() {
	infos := core.AvailableServiceUpdates()
	if len(infos) == 0 {
		return
	}
	run := func(names []string) {
		setUpdateApplyBusy(true)
		go core.ApplyServiceUpdates(names)
	}
	var items []*fyne.MenuItem
	all := make([]string, 0, len(infos))
	for _, info := range infos {
		info := info
		all = append(all, info.Name)
		current := core.CompactServiceVersion(info.Name, info.Current.Version)
		latest := core.CompactServiceVersion(info.Name, info.Latest.Version)
		label := info.Name + "  " + latest
		if current != "" && latest != "" {
			label = fmt.Sprintf("%s   %s → %s", info.Name, current, latest)
		}
		items = append(items, fyne.NewMenuItem(label, func() { run([]string{info.Name}) }))
	}
	items = append(items, fyne.NewMenuItemSeparator(),
		fyne.NewMenuItemWithIcon(fmt.Sprintf("Update all (%d)", len(infos)), theme.DownloadIcon(), func() { run(all) }))
	showMenuBelow(fyne.NewMenu("Updates", items...), svcUI.updateBtn)
}
