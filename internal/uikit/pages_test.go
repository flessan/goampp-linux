package uikit

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestProjectsPageUsesReferenceCards(t *testing.T) {
	test.NewTempApp(t)
	page := ProjectsPage(widget.NewEntry(), widget.NewLabel("list"), "description")
	if _, ok := page.(*fyne.Container); !ok {
		t.Fatalf("projects page = %T, want border container", page)
	}
	if EmptyProjects() == nil {
		t.Fatal("project empty state missing")
	}
}

func TestStartupAndSettingsPages(t *testing.T) {
	test.NewTempApp(t)
	platform := widget.NewCheck("launch", nil)
	services := widget.NewLabel("services")
	if StartupPage("Windows", platform, services, "Services", "description") == nil {
		t.Fatal("startup page missing")
	}
	page := SettingsPage(widget.NewLabel("appearance"), widget.NewLabel("paths"),
		widget.NewLabel("tools"), widget.NewLabel("about"), widget.NewLabel("behavior"))
	if _, ok := page.(*container.Scroll); !ok {
		t.Fatalf("settings page = %T, want scroll", page)
	}
}
