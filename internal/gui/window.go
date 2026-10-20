//go:build windows

// Package gui is the Fyne desktop front end.
package gui

import (
	"fmt"
	"image/color"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"goampp/internal/core"
	"goampp/internal/packages"
)

const appID = "id.imtaqin.goampp"

type pageDef struct {
	key, title string
	icon       fyne.Resource
	build      func() (actions fyne.CanvasObject, body fyne.CanvasObject)
	onShow     func()

	actions, body fyne.CanvasObject
	nav           *navItem
}

// gui holds the widgets that outlive a single build. Everything is rebuilt
// when the theme changes, so nothing here caches colours.
var gui struct {
	pages   []*pageDef
	current string

	pageHost      *fyne.Container
	headerTitle   *canvas.Text
	headerActions *fyne.Container

	stackStat *canvas.Text
	stackDot  *canvas.Circle

	statusText  *widget.Label
	statusDot   *canvas.Circle
	progressBar *widget.ProgressBar
	busy        bool

	logList     *widget.List
	logLines    []string
	logBody     fyne.CanvasObject
	logFollow   bool
	logDirty    atomic.Bool
	logToggle   *widget.Button
	shownHidden bool
}

// guiDo runs fn on the GUI goroutine. It is safe to call from anywhere,
// including before the window exists (the call is then dropped).
func guiDo(fn func()) {
	if core.App == nil || core.App.Win == nil {
		return
	}
	fyne.Do(fn)
}

// guiStatus shows a short message in the bottom bar until the next log line
// or status replaces it.
func guiStatus(text string) {
	guiDo(func() {
		if gui.statusText != nil && !gui.busy {
			gui.statusText.Importance = widget.MediumImportance
			gui.statusText.SetText(text)
		}
	})
}

func Run(startHidden bool, onReady func()) {
	a := fyneapp.NewWithID(appID)
	a.Settings().SetTheme(newGoamppTheme(a.Preferences().StringWithFallback(themePrefKey, themeSystem)))
	if logo := logoResource(); logo != nil {
		a.SetIcon(logo)
	}

	w := a.NewWindow("GoAMPP")
	core.App.Fyne, core.App.Win = a, w
	core.UI = core.UIHooks{
		Do:              guiDo,
		Status:          guiStatus,
		RefreshServices: refreshServiceList,
		Progress:        guiDownloadProgress,
		UpdateCheckDone: func() { refreshServiceList(); setUpdateRefreshBusy(false) },
		UpdateApplyDone: func() { refreshServiceList(); setUpdateApplyBusy(false) },
	}

	if err := packages.LoadPersistedLatestCatalog(core.App.BaseDir); err != nil {
		core.App.AppendLog("updates: load cached versions: " + err.Error())
	}

	gui.logFollow = true
	rebuildContent()
	w.Resize(fyne.NewSize(1080, 680))
	w.CenterOnScreen()

	w.SetCloseIntercept(func() {
		w.Hide()
		if !gui.shownHidden {
			gui.shownHidden = true
			a.SendNotification(fyne.NewNotification("GoAMPP is still running",
				"Services keep running in the background. Use the tray icon to reopen or quit."))
		}
	})
	w.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: fyne.KeyModifierControl}, func(fyne.Shortcut) {
		if gui.current == "editor" {
			saveEditorBuffer()
		}
	})

	installTray(a, w)

	// Theme follows Windows when set to System: redraw on any settings change.
	a.Settings().AddListener(func(fyne.Settings) { fyne.Do(rebuildContent) })

	core.App.LogMu.Lock()
	core.App.OnLog = func() { gui.logDirty.Store(true) }
	core.App.LogMu.Unlock()
	go logPump()

	a.Lifecycle().SetOnStarted(func() {
		go onReady()
		go warmVersionInfo(a)
	})
	a.Lifecycle().SetOnStopped(core.StopAllServices)

	if startHidden {
		a.Run()
	} else {
		w.ShowAndRun()
	}
}

// warmVersionInfo resolves runtime branches right away and checks every
// package for updates at most twice a day, so rows show real versions and
// update badges without the user pressing anything.
func warmVersionInfo(a fyne.App) {
	if err := packages.RefreshVariantCatalog(false); err != nil {
		core.App.AppendLog("downloads: latest runtime branch lookup: " + err.Error())
	} else if err := packages.PersistLatestCatalog(core.App.BaseDir, nil); err != nil {
		core.App.AppendLog("updates: save cached runtime versions: " + err.Error())
	}
	core.UiRefreshServices()

	const key = "lastUpdateCheck"
	last := time.Unix(int64(a.Preferences().Int(key)), 0)
	if time.Since(last) < 12*time.Hour {
		return
	}
	a.Preferences().SetInt(key, int(time.Now().Unix()))
	core.UiDo(func() { setUpdateRefreshBusy(true) })
	core.RefreshAllServiceUpdates()
}

func logPump() {
	t := time.NewTicker(200 * time.Millisecond)
	defer t.Stop()
	for range t.C {
		if !gui.logDirty.Swap(false) {
			continue
		}
		lines := core.App.LogSnapshot()
		fyne.Do(func() { showLogLines(lines) })
	}
}

func showLogLines(lines []string) {
	gui.logLines = lines
	if gui.logList != nil {
		gui.logList.Refresh()
		if gui.logFollow && len(lines) > 0 {
			gui.logList.ScrollToBottom()
		}
	}
	if gui.statusText != nil && !gui.busy && len(lines) > 0 {
		last := lines[len(lines)-1]
		gui.statusText.Importance = logImportance(last)
		gui.statusText.SetText(last)
	}
}

func rebuildContent() {
	if core.App.Win == nil {
		return
	}
	current := gui.current
	if current == "" {
		current = "services"
	}
	gui.pages = []*pageDef{
		{key: "services", title: "Services", icon: theme.ComputerIcon(), build: buildServicesPage},
		{key: "projects", title: "Projects", icon: theme.FolderIcon(), build: buildProjectsPage},
		{key: "vhosts", title: "Virtual Hosts", icon: theme.HomeIcon(), build: buildVhostsPage},
		{key: "editor", title: "Config Editor", icon: theme.DocumentCreateIcon(), build: buildEditorPage, onShow: populateEditorDropdown},
		{key: "startup", title: "Startup", icon: theme.MediaPlayIcon(), build: buildStartupPage, onShow: refreshStartupPage},
		{key: "settings", title: "Settings", icon: theme.SettingsIcon(), build: buildSettingsPage},
	}

	gui.headerTitle = boldText("", 17)
	gui.headerActions = container.NewHBox()

	gui.pageHost = container.NewStack()
	for _, p := range gui.pages {
		p.actions, p.body = p.build()
		p.body.Hide()
		gui.pageHost.Add(p.body)
	}

	header := container.NewBorder(nil, nil, container.NewCenter(gui.headerTitle), gui.headerActions)
	main := container.NewBorder(
		container.New(layout.NewCustomPaddedLayout(12, 8, 18, 18), header),
		buildBottomBar(),
		nil, nil,
		container.New(layout.NewCustomPaddedLayout(0, 10, 18, 18), gui.pageHost),
	)

	bg := canvas.NewRectangle(themeColor(theme.ColorNameBackground))
	core.App.Win.SetContent(container.NewBorder(nil, nil, buildSidebar(), nil, container.NewStack(bg, main)))

	gui.current = ""
	showPage(current)
	refreshServiceList()
	refreshVhostList()
	refreshProjectList()
	showLogLines(core.App.LogSnapshot())
}

func showPage(key string) {
	for _, p := range gui.pages {
		if p.key != key {
			p.body.Hide()
			p.nav.setSelected(false)
			continue
		}
		p.body.Show()
		p.nav.setSelected(true)
		gui.headerTitle.Text = p.title
		gui.headerTitle.Refresh()
		gui.headerActions.Objects = nil
		if p.actions != nil {
			gui.headerActions.Objects = []fyne.CanvasObject{p.actions}
		}
		gui.headerActions.Refresh()
		if gui.current != key && p.onShow != nil {
			p.onShow()
		}
	}
	gui.current = key
}

func buildSidebar() fyne.CanvasObject {
	logo := fyne.CanvasObject(letterAvatar("GoAMPP", 22))
	if res := logoResource(); res != nil {
		img := canvas.NewImageFromResource(res)
		img.FillMode = canvas.ImageFillContain
		img.ScaleMode = canvas.ImageScaleSmooth
		logo = container.NewGridWrap(fyne.NewSize(22, 22), img)
	}
	brand := container.NewHBox(logo, container.NewCenter(boldText("GoAMPP", 14)))

	nav := container.NewVBox()
	for _, p := range gui.pages {
		p := p
		p.nav = newNavItem(p.title, p.icon, func() { showPage(p.key) })
		nav.Add(p.nav)
	}

	gui.stackDot = canvas.NewCircle(themeColor(colorMuted))
	gui.stackStat = mutedText("", 11.5)
	stat := container.NewHBox(hSpace(8), container.NewCenter(container.NewGridWrap(fyne.NewSize(6, 6), gui.stackDot)), gui.stackStat)

	start := lowButton("Start stack", theme.NewSuccessThemedResource(theme.MediaPlayIcon()), func() { go core.EnsureStackEssentials() })
	start.Alignment = widget.ButtonAlignLeading
	stop := lowButton("Stop all", theme.NewErrorThemedResource(theme.MediaStopIcon()), func() { go core.StopAllServices() })
	stop.Alignment = widget.ButtonAlignLeading

	body := container.NewBorder(
		container.NewVBox(container.New(layout.NewCustomPaddedLayout(0, 0, 6, 0), brand), vSpace(14), nav),
		container.NewVBox(stat, start, stop, vSpace(4),
			container.New(layout.NewCustomPaddedLayout(0, 0, 8, 0), mutedText(core.Version, 10.5))),
		nil, nil,
	)
	bg := canvas.NewRectangle(themeColor(colorSidebar))
	edge := canvas.NewRectangle(themeColor(theme.ColorNameSeparator))
	return container.NewBorder(nil, nil, nil, container.NewGridWrap(fyne.NewSize(1, 1), edge),
		container.NewStack(bg, minSized(container.New(layout.NewCustomPaddedLayout(14, 10, 10, 10), body), 184, 0)))
}

func refreshStackSummary() {
	if gui.stackStat == nil {
		return
	}
	running, installed := core.RunningServiceCount()
	gui.stackStat.Text = fmt.Sprintf("%d running · %d installed", running, installed)
	gui.stackStat.Refresh()
	if running > 0 {
		gui.stackDot.FillColor = colSuccess
	} else {
		gui.stackDot.FillColor = themeColor(colorMuted)
	}
	gui.stackDot.Refresh()
}

// ---- bottom bar: status / progress line + collapsible log ----

func buildBottomBar() fyne.CanvasObject {
	gui.statusText = widget.NewLabel("Ready")
	gui.statusText.SizeName = theme.SizeNameCaptionText
	gui.statusText.TextStyle = fyne.TextStyle{Monospace: true}
	gui.statusText.Truncation = fyne.TextTruncateEllipsis
	gui.statusText.Importance = widget.LowImportance
	gui.statusDot = canvas.NewCircle(themeColor(theme.ColorNamePrimary))
	gui.progressBar = widget.NewProgressBar()
	gui.progressBar.TextFormatter = func() string { return "" }
	gui.progressBar.Hide()

	gui.logList = widget.NewList(
		func() int { return len(gui.logLines) },
		func() fyne.CanvasObject {
			t := canvas.NewText("", themeColor(theme.ColorNameForeground))
			t.TextStyle = fyne.TextStyle{Monospace: true}
			t.TextSize = 11.5
			return t
		},
		func(id widget.ListItemID, o fyne.CanvasObject) {
			t := o.(*canvas.Text)
			if id < len(gui.logLines) {
				line := gui.logLines[id]
				t.Text = line
				t.Color = logColor(line)
				t.Refresh()
			}
		},
	)
	gui.logList.HideSeparators = true
	logBg := canvas.NewRectangle(themeColor(colorCard))
	logBg.CornerRadius = 6
	logBg.StrokeColor = themeColor(colorCardBorder)
	logBg.StrokeWidth = 1
	gui.logBody = minSized(container.NewStack(logBg, container.New(layout.NewCustomPaddedLayout(2, 2, 6, 6), gui.logList)), 0, 150)
	gui.logBody.Hide()

	follow := widget.NewCheck("Follow", func(v bool) { gui.logFollow = v })
	follow.SetChecked(gui.logFollow)
	clear := lowButton("", theme.ContentClearIcon(), func() {
		core.App.ClearLog()
		showLogLines(nil)
	})
	copyBtn := lowButton("", theme.ContentCopyIcon(), func() {
		core.App.Fyne.Clipboard().SetContent(strings.Join(core.App.LogSnapshot(), "\r\n"))
		core.SetStatus("Log copied to clipboard")
	})
	logsDir := lowButton("", theme.FolderOpenIcon(), func() { core.OpenFolder(filepath.Join(core.App.BaseDir, "logs")) })
	tools := container.NewHBox(follow, clear, copyBtn, logsDir)
	tools.Hide()
	gui.logToggle = lowButton("Log", theme.MenuDropUpIcon(), nil)
	gui.logToggle.IconPlacement = widget.ButtonIconTrailingText
	gui.logToggle.OnTapped = func() {
		if gui.logBody.Visible() {
			gui.logBody.Hide()
			tools.Hide()
			gui.logToggle.SetIcon(theme.MenuDropUpIcon())
		} else {
			gui.logBody.Show()
			tools.Show()
			gui.logToggle.SetIcon(theme.MenuDropDownIcon())
			gui.logList.ScrollToBottom()
		}
	}

	dot := container.NewCenter(container.NewGridWrap(fyne.NewSize(6, 6), gui.statusDot))
	line := container.NewBorder(nil, nil,
		container.NewHBox(dot, hSpace(2)),
		container.NewHBox(minSized(gui.progressBar, 160, 0), tools, gui.logToggle),
		gui.statusText)

	edge := canvas.NewRectangle(themeColor(theme.ColorNameSeparator))
	edge.SetMinSize(fyne.NewSize(1, 1))
	return container.NewBorder(edge, nil, nil, nil,
		container.New(layout.NewCustomPaddedLayout(4, 4, 18, 12), container.NewVBox(gui.logBody, line)))
}

func logImportance(line string) widget.Importance {
	switch logColor(line) {
	case colError:
		return widget.DangerImportance
	case colWarning:
		return widget.WarningImportance
	case colSuccess:
		return widget.SuccessImportance
	}
	return widget.LowImportance
}

func logColor(line string) color.Color {
	l := strings.ToLower(line)
	switch {
	case strings.Contains(l, "failed") || strings.Contains(l, "error") || strings.Contains(l, "http 4") || strings.Contains(l, "http 5"):
		return colError
	case strings.Contains(l, "warn") || strings.Contains(l, "skipping"):
		return colWarning
	case strings.Contains(l, "installed ") || strings.Contains(l, " started") || strings.Contains(l, "switched to") || strings.Contains(l, "applied"):
		return colSuccess
	}
	return themeColor(colorMuted)
}

// guiDownloadProgress is the ProgressFunc handed to every download. It feeds
// both the bottom bar and the row of the service being installed.
func guiDownloadProgress(stage, name string, done, total int64) {
	var (
		text     string
		fraction float64
		active   = true
	)
	switch stage {
	case "starting":
		text = fmt.Sprintf("Preparing %s...", name)
	case "downloading":
		if total > 0 {
			fraction = float64(done) / float64(total)
			text = fmt.Sprintf("Downloading %s  %.1f / %.1f MB", name, float64(done)/(1<<20), float64(total)/(1<<20))
		} else {
			text = fmt.Sprintf("Downloading %s  %.1f MB", name, float64(done)/(1<<20))
		}
	case "extracting":
		if total > 0 {
			fraction = float64(done) / float64(total)
			text = fmt.Sprintf("Extracting %s  %d / %d files", name, done, total)
		} else {
			text = fmt.Sprintf("Extracting %s...", name)
			fraction = 0.5
		}
	case "post-install":
		text = fmt.Sprintf("Configuring %s...", name)
		fraction = 1
	case "done":
		text = fmt.Sprintf("Installed %s", name)
		active = false
	case "idle":
		active = false
	default:
		text = stage
	}
	core.UiDo(func() {
		setCardProgress(name, active, fraction)
		if gui.statusText == nil {
			return
		}
		gui.busy = active
		if !active {
			gui.progressBar.Hide()
			if text != "" {
				gui.statusText.Importance = widget.SuccessImportance
				gui.statusText.SetText(text)
			}
			refreshServiceList()
			return
		}
		gui.statusText.Importance = widget.MediumImportance
		gui.statusText.SetText(text)
		gui.progressBar.SetValue(fraction)
		gui.progressBar.Show()
	})
}

// ---- tray ----

func installTray(a fyne.App, w fyne.Window) {
	desk, ok := a.(desktop.App)
	if !ok {
		return
	}
	show := fyne.NewMenuItem("Show GoAMPP", func() { w.Show(); w.RequestFocus() })
	start := fyne.NewMenuItem("Start stack", func() { go core.EnsureStackEssentials() })
	stop := fyne.NewMenuItem("Stop all services", func() { go core.StopAllServices() })
	auto := fyne.NewMenuItem("Launch with Windows", nil)
	auto.Checked = core.IsAutoStartEnabled()
	quit := fyne.NewMenuItem("Quit", quitApp)
	quit.IsQuit = true

	menu := fyne.NewMenu("GoAMPP", show, fyne.NewMenuItemSeparator(), start, stop,
		fyne.NewMenuItemSeparator(), auto, fyne.NewMenuItemSeparator(), quit)
	auto.Action = func() {
		core.ToggleAutoStart()
		auto.Checked = core.IsAutoStartEnabled()
		menu.Refresh()
	}
	desk.SetSystemTrayMenu(menu)
	if logo := logoResource(); logo != nil {
		desk.SetSystemTrayIcon(logo)
	}
	desk.SetSystemTrayWindow(w)
}

func quitApp() {
	core.SetStatus("Stopping services...")
	go func() {
		core.StopAllServices()
		fyne.Do(core.App.Fyne.Quit)
	}()
}
