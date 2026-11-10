//go:build linux

package linuxapp

import (
	"image/color"

	"fyne.io/fyne/v2/theme"

	"goampp/internal/uikit"
)

const (
	linuxColorCard       = uikit.ColorNameCard
	linuxColorCardBorder = uikit.ColorNameCardBorder
	linuxColorSidebar    = uikit.ColorNameSidebar
	linuxColorMuted      = uikit.ColorNameMuted
	linuxColorNavActive  = uikit.ColorNameNavActive
	linuxColorLogBg      = uikit.ColorNameLogBg
)

type linuxGoamppTheme = uikit.Theme

func newLinuxGoamppTheme(preference string) *linuxGoamppTheme {
	return uikit.NewTheme(preference)
}

// The recoloring pass uses the same exported palette as the shared Fyne theme.
var (
	linuxDarkPalette  = uikit.PaletteForVariant(theme.VariantDark)
	linuxLightPalette = uikit.PaletteForVariant(theme.VariantLight)
)

func linuxHex(value uint32) color.NRGBA {
	return uikit.Hex(value)
}

func linuxHexA(value uint32, alpha uint8) color.NRGBA {
	return uikit.HexA(value, alpha)
}
