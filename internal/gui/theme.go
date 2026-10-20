//go:build windows

package gui

import (
	"image/color"
	"os"
	"path/filepath"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Extra colour names used by GoAMPP's own widgets (cards, sidebar, pills).
const (
	colorCard       fyne.ThemeColorName = "goampp.card"
	colorCardBorder fyne.ThemeColorName = "goampp.cardBorder"
	colorSidebar    fyne.ThemeColorName = "goampp.sidebar"
	colorMuted      fyne.ThemeColorName = "goampp.muted"
	colorNavActive  fyne.ThemeColorName = "goampp.navActive"
	colorLogBg      fyne.ThemeColorName = "goampp.logBg"
)

const (
	themePrefKey = "theme"
	themeSystem  = "System"
	themeLight   = "Light"
	themeDark    = "Dark"
)

type palette struct {
	bg, sidebar, card, border, input, button, hover, pressed, fg, muted,
	primary, navActive, separator, menu, overlay, header, scroll, shadow,
	selection, logBg, placeholder, disabled, disabledBtn color.Color
}

func hex(v uint32) color.NRGBA {
	return color.NRGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 0xff}
}

func hexA(v uint32, a uint8) color.NRGBA {
	c := hex(v)
	c.A = a
	return c
}

// A neutral zinc palette with a single blue accent: colour is kept for state
// (running, error, update) so it carries meaning instead of decoration.
var darkPalette = palette{
	bg:          hex(0x0c0c0e),
	sidebar:     hex(0x0f0f12),
	card:        hex(0x131316),
	border:      hex(0x222226),
	input:       hex(0x18181c),
	button:      hex(0x1c1c21),
	hover:       hexA(0xffffff, 0x0c),
	pressed:     hexA(0xffffff, 0x18),
	fg:          hex(0xe4e4e7),
	muted:       hex(0x85858f),
	primary:     hex(0x4f8cff),
	navActive:   hexA(0xffffff, 0x0e),
	separator:   hex(0x1f1f23),
	menu:        hex(0x17171b),
	overlay:     hex(0x131316),
	header:      hex(0x131316),
	scroll:      hexA(0xffffff, 0x26),
	shadow:      hexA(0x000000, 0x66),
	selection:   hexA(0x4f8cff, 0x40),
	logBg:       hex(0x0c0c0e),
	placeholder: hex(0x5b5b64),
	disabled:    hex(0x55555e),
	disabledBtn: hex(0x16161a),
}

var lightPalette = palette{
	bg:          hex(0xfafafa),
	sidebar:     hex(0xf4f4f5),
	card:        hex(0xffffff),
	border:      hex(0xe4e4e7),
	input:       hex(0xffffff),
	button:      hex(0xf4f4f5),
	hover:       hexA(0x000000, 0x08),
	pressed:     hexA(0x000000, 0x12),
	fg:          hex(0x18181b),
	muted:       hex(0x71717a),
	primary:     hex(0x2563eb),
	navActive:   hexA(0x000000, 0x08),
	separator:   hex(0xe9e9ec),
	menu:        hex(0xffffff),
	overlay:     hex(0xffffff),
	header:      hex(0xf4f4f5),
	scroll:      hexA(0x000000, 0x26),
	shadow:      hexA(0x000000, 0x22),
	selection:   hexA(0x2563eb, 0x30),
	logBg:       hex(0xfafafa),
	placeholder: hex(0xa1a1aa),
	disabled:    hex(0xa1a1aa),
	disabledBtn: hex(0xf4f4f5),
}

var (
	colSuccess = hex(0x22c55e)
	colWarning = hex(0xf59e0b)
	colError   = hex(0xef4444)
)

type goamppTheme struct {
	// forced is nil to follow Windows, or the variant the user picked.
	forced *fyne.ThemeVariant
}

func newGoamppTheme(pref string) *goamppTheme {
	t := &goamppTheme{}
	switch pref {
	case themeLight:
		v := theme.VariantLight
		t.forced = &v
	case themeDark:
		v := theme.VariantDark
		t.forced = &v
	}
	return t
}

func (t *goamppTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if t.forced != nil {
		variant = *t.forced
	}
	p := &darkPalette
	if variant == theme.VariantLight {
		p = &lightPalette
	}
	switch name {
	case theme.ColorNameBackground:
		return p.bg
	case theme.ColorNameButton:
		return p.button
	case theme.ColorNameDisabledButton:
		return p.disabledBtn
	case theme.ColorNameDisabled:
		return p.disabled
	case theme.ColorNameForeground:
		return p.fg
	case theme.ColorNameHover:
		return p.hover
	case theme.ColorNamePressed:
		return p.pressed
	case theme.ColorNameInputBackground:
		return p.input
	case theme.ColorNameInputBorder:
		return p.border
	case theme.ColorNamePrimary, theme.ColorNameFocus, theme.ColorNameHyperlink:
		return p.primary
	case theme.ColorNameForegroundOnPrimary, theme.ColorNameForegroundOnError,
		theme.ColorNameForegroundOnSuccess:
		return color.White
	case theme.ColorNameForegroundOnWarning:
		return hex(0x1b1200)
	case theme.ColorNameSelection:
		return p.selection
	case theme.ColorNameSeparator:
		return p.separator
	case theme.ColorNameMenuBackground:
		return p.menu
	case theme.ColorNameOverlayBackground:
		return p.overlay
	case theme.ColorNameHeaderBackground:
		return p.header
	case theme.ColorNameScrollBar:
		return p.scroll
	case theme.ColorNameShadow:
		return p.shadow
	case theme.ColorNamePlaceHolder:
		return p.placeholder
	case theme.ColorNameSuccess:
		return colSuccess
	case theme.ColorNameWarning:
		return colWarning
	case theme.ColorNameError:
		return colError
	case colorCard:
		return p.card
	case colorCardBorder:
		return p.border
	case colorSidebar:
		return p.sidebar
	case colorMuted:
		return p.muted
	case colorNavActive:
		return p.navActive
	case colorLogBg:
		return p.logBg
	}
	return theme.DefaultTheme().Color(name, variant)
}

func (t *goamppTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *goamppTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 12.5
	case theme.SizeNameCaptionText:
		return 11
	case theme.SizeNameHeadingText:
		return 17
	case theme.SizeNameSubHeadingText:
		return 14
	case theme.SizeNamePadding:
		return 4
	case theme.SizeNameInnerPadding:
		return 6
	case theme.SizeNameLineSpacing:
		return 3
	case theme.SizeNameInlineIcon:
		return 16
	case theme.SizeNameInputRadius, theme.SizeNameButtonRadius:
		return 6
	case theme.SizeNameSelectionRadius:
		return 5
	case theme.SizeNameCardRadius, theme.SizeNameDialogRadius:
		return 8
	case theme.SizeNamePopupRadius, theme.SizeNameMenuRadius:
		return 8
	case theme.SizeNameScrollBar:
		return 8
	case theme.SizeNameScrollBarSmall:
		return 3
	case theme.SizeNameScrollBarRadius:
		return 3
	case theme.SizeNameSeparatorThickness:
		return 1
	case theme.SizeNameInputBorder:
		return 1
	}
	return theme.DefaultTheme().Size(name)
}

func (t *goamppTheme) Font(style fyne.TextStyle) fyne.Resource {
	if f := systemFont(style); f != nil {
		return f
	}
	return theme.DefaultTheme().Font(style)
}

// systemFont uses Segoe UI (and Cascadia/Consolas for code) from the Windows
// font folder so the panel reads like a native app. Missing files fall back to
// Fyne's bundled font.
var (
	fontOnce  sync.Once
	fontCache map[string]fyne.Resource
)

func systemFont(style fyne.TextStyle) fyne.Resource {
	fontOnce.Do(func() {
		fontCache = map[string]fyne.Resource{}
		dir := filepath.Join(os.Getenv("WINDIR"), "Fonts")
		load := func(key string, files ...string) {
			for _, f := range files {
				data, err := os.ReadFile(filepath.Join(dir, f))
				if err == nil {
					fontCache[key] = fyne.NewStaticResource(f, data)
					return
				}
			}
		}
		load("regular", "segoeui.ttf")
		load("bold", "seguisb.ttf", "segoeuib.ttf")
		load("italic", "segoeuii.ttf")
		load("mono", "CascadiaMono.ttf", "consola.ttf")
	})
	switch {
	case style.Monospace:
		return fontCache["mono"]
	case style.Bold:
		return fontCache["bold"]
	case style.Italic:
		return fontCache["italic"]
	case style.Symbol:
		return nil
	}
	return fontCache["regular"]
}

// themeColor resolves one of the theme colours for the current variant.
func themeColor(name fyne.ThemeColorName) color.Color {
	return theme.Color(name)
}
