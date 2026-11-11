package uikit

import (
	"testing"

	"fyne.io/fyne/v2"

	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestSystemThemeUsesTheRequestedVariantPalette(t *testing.T) {
	system := NewTheme("System")

	for _, variant := range []fyne.ThemeVariant{theme.VariantLight, theme.VariantDark} {
		palette := PaletteForVariant(variant)
		for name, want := range map[fyne.ThemeColorName]any{
			theme.ColorNameBackground: palette.Background,
			theme.ColorNameForeground: palette.Foreground,
			ColorNameCard:             palette.Card,
			ColorNameCardBorder:       palette.Border,
			ColorNameSidebar:          palette.Sidebar,
			ColorNameMuted:            palette.Muted,
			ColorNameNavActive:        palette.NavActive,
			ColorNameLogBg:            palette.LogBackground,
		} {
			if got := system.Color(name, variant); got != want {
				t.Errorf("System theme color %q for variant %v = %v, want %v", name, variant, got, want)
			}
		}
	}
}

func TestDarkPreferenceForcesDarkPalette(t *testing.T) {
	dark := NewTheme("Dark")
	want := PaletteForVariant(theme.VariantDark)

	for _, variant := range []fyne.ThemeVariant{theme.VariantLight, theme.VariantDark} {
		if got := dark.Color(theme.ColorNameBackground, variant); got != want.Background {
			t.Errorf("Dark theme background for variant %v = %v, want %v", variant, got, want.Background)
		}
		if got := dark.Color(ColorNameCard, variant); got != want.Card {
			t.Errorf("Dark theme card for variant %v = %v, want %v", variant, got, want.Card)
		}
	}
}

func TestSharedThemeSizingTokens(t *testing.T) {
	dark := NewTheme("Dark")
	checks := map[fyne.ThemeSizeName]float32{
		theme.SizeNameText:               12.5,
		theme.SizeNameCaptionText:        11,
		theme.SizeNameInputRadius:        6,
		theme.SizeNameButtonRadius:       6,
		theme.SizeNameCardRadius:         8,
		theme.SizeNameScrollBar:          8,
		theme.SizeNameSeparatorThickness: 1,
	}
	for name, want := range checks {
		if got := dark.Size(name); got != want {
			t.Errorf("shared theme size %q = %v, want %v", name, got, want)
		}
	}
}

func TestLightPreferenceForcesLightPalette(t *testing.T) {
	light := NewTheme("Light")
	want := PaletteForVariant(theme.VariantLight)
	for _, variant := range []fyne.ThemeVariant{theme.VariantLight, theme.VariantDark} {
		if got := light.Color(theme.ColorNameBackground, variant); got != want.Background {
			t.Errorf("Light theme background for variant %v = %v, want %v", variant, got, want.Background)
		}
	}
}

func TestVirtualHostsPageUsesReferenceSplit(t *testing.T) {
	test.NewTempApp(t)
	list := widget.NewLabel("domains")
	editor := widget.NewLabel("editor")
	page := VirtualHostsPage(list, editor)
	split, ok := page.(*container.Split)
	if !ok {
		t.Fatalf("page type = %T, want split", page)
	}
	if split.Offset != 0.52 || split.Trailing != editor {
		t.Fatalf("unexpected split offset or editor: %v", split.Offset)
	}
}

func TestConfigEditorPageKeepsControlsInHeader(t *testing.T) {
	test.NewTempApp(t)
	files := widget.NewSelect(nil, nil)
	reload := widget.NewButton("Reload", nil)
	save := widget.NewButton("Save", nil)
	actions, body := ConfigEditorPage(files, []fyne.CanvasObject{reload, save}, widget.NewLabel("path"), nil, nil, widget.NewMultiLineEntry())
	header, ok := actions.(*fyne.Container)
	if !ok || len(header.Objects) != 3 || header.Objects[1] != reload || header.Objects[2] != save {
		t.Fatal("editor actions are not in the reference order")
	}
	if body == nil {
		t.Fatal("editor body is missing")
	}
}
