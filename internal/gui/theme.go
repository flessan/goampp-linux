//go:build windows

package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"goampp/internal/uikit"
)

const (
	colorCard       = uikit.ColorNameCard
	colorCardBorder = uikit.ColorNameCardBorder
	colorSidebar    = uikit.ColorNameSidebar
	colorMuted      = uikit.ColorNameMuted
	colorNavActive  = uikit.ColorNameNavActive
	colorLogBg      = uikit.ColorNameLogBg
)

const (
	themePrefKey = "theme"
	themeSystem  = "System"
	themeDark    = "Dark"
)

type goamppTheme = uikit.Theme

func newGoamppTheme(preference string) *goamppTheme {
	return uikit.NewTheme(preference)
}

// hex helpers remain available to the Windows-only custom avatar widgets.
func hex(value uint32) color.NRGBA {
	return uikit.Hex(value)
}

func hexA(value uint32, alpha uint8) color.NRGBA {
	return uikit.HexA(value, alpha)
}

var (
	colSuccess color.Color = theme.Color(theme.ColorNameSuccess)
	colWarning color.Color = theme.Color(theme.ColorNameWarning)
	colError   color.Color = theme.Color(theme.ColorNameError)
)
