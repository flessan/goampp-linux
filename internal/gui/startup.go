//go:build windows

package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"goampp/internal/core"
	"goampp/internal/uikit"
)

var startupUI struct {
	windows *widget.Check
	list    *fyne.Container
}

func buildStartupPage() (fyne.CanvasObject, fyne.CanvasObject) {
	startupUI.windows = widget.NewCheck("Launch GoAMPP when I sign in to Windows (starts minimised to the tray)", func(v bool) {
		if v == core.IsAutoStartEnabled() {
			return
		}
		core.ToggleAutoStart()
	})

	startupUI.list = container.NewGridWrap(fyne.NewSize(300, 56))
	refreshStartupPage()
	return nil, uikit.StartupPage("Windows", startupUI.windows, container.NewVScroll(startupUI.list),
		"Services started by GoAMPP", "Ticked services start every time GoAMPP opens. Only installed services are listed.")
}

func refreshStartupPage() {
	if startupUI.list == nil {
		return
	}
	startupUI.windows.SetChecked(core.IsAutoStartEnabled())

	startupUI.list.Objects = nil
	for _, ms := range core.App.Services {
		if !core.StartupServiceEligible(ms) {
			continue
		}
		name := ms.Conf.Name
		label := name
		if ms.Conf.Port > 0 {
			label += fmt.Sprintf("  :%d", ms.Conf.Port)
		}
		check := widget.NewCheck(label, nil)
		check.SetChecked(core.AutoStartContains(name))
		check.OnChanged = func(v bool) {
			if err := core.SetServiceAutoStart(name, v); err != nil {
				core.App.AppendLog("startup services: " + err.Error())
				return
			}
			state := "skip"
			if v {
				state = "start"
			}
			core.App.AppendLog(fmt.Sprintf("startup services: %s → %s", name, state))
		}
		startupUI.list.Add(uikit.StartupServiceCard(serviceAvatar(name, 24), check, nil))
	}
	if len(startupUI.list.Objects) == 0 {
		startupUI.list.Add(mutedText("No installed services yet — install some from Services.", 13))
	}
	startupUI.list.Refresh()
}
