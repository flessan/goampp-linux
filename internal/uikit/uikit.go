// Package uikit contains the shared desktop UI primitives used by the
// Windows and Linux front ends. Platform-specific behavior belongs in the
// platform front ends; visual structure and spacing live here.
package uikit

import (
	"image/color"
	"strings"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	ColorNameCard       fyne.ThemeColorName = "goampp.card"
	ColorNameCardBorder fyne.ThemeColorName = "goampp.cardBorder"
	ColorNameSidebar    fyne.ThemeColorName = "goampp.sidebar"
	ColorNameMuted      fyne.ThemeColorName = "goampp.muted"
	ColorNameNavActive  fyne.ThemeColorName = "goampp.navActive"
	ColorNameLogBg      fyne.ThemeColorName = "goampp.logBg"
)

// Pad applies the shared custom padding order: top, bottom, left, right.
func Pad(obj fyne.CanvasObject, top, bottom, left, right float32) fyne.CanvasObject {
	return container.New(layout.NewCustomPaddedLayout(top, bottom, left, right), obj)
}

// Surface wraps content in the shared rounded, bordered application surface.
func Surface(content fyne.CanvasObject, top, bottom, left, right float32) fyne.CanvasObject {
	background := canvas.NewRectangle(theme.Color(ColorNameCard))
	background.CornerRadius = theme.Size(theme.SizeNameCardRadius)
	background.StrokeColor = theme.Color(ColorNameCardBorder)
	background.StrokeWidth = 1
	return container.NewStack(background, Pad(content, top, bottom, left, right))
}

// Gap creates a transparent spacing object.
func Gap(width, height float32) fyne.CanvasObject {
	gap := canvas.NewRectangle(color.Transparent)
	gap.SetMinSize(fyne.NewSize(width, height))
	return gap
}

// MinSize keeps an object at least the requested size without changing its
// natural rendering or alignment.
func MinSize(obj fyne.CanvasObject, width, height float32) fyne.CanvasObject {
	return container.NewStack(Gap(width, height), obj)
}

func BoldText(value string, size float32) *canvas.Text {
	text := canvas.NewText(value, theme.Color(theme.ColorNameForeground))
	text.TextStyle = fyne.TextStyle{Bold: true}
	text.TextSize = size
	return text
}

func MutedText(value string, size float32) *canvas.Text {
	text := canvas.NewText(value, theme.Color(ColorNameMuted))
	text.TextSize = size
	return text
}

// TitledCard is the Windows card heading, spacing and surface treatment.
func TitledCard(title, subtitle string, actions, body fyne.CanvasObject) fyne.CanvasObject {
	head := container.NewVBox(BoldText(title, 13))
	if subtitle != "" {
		head.Add(MutedText(subtitle, 11.5))
	}
	var top fyne.CanvasObject = head
	if actions != nil {
		top = container.NewBorder(nil, nil, nil, container.NewCenter(actions), head)
	}
	if body == nil {
		return Surface(top, 10, 10, 12, 12)
	}
	return Surface(container.NewBorder(container.NewVBox(top, Gap(1, 4)), nil, nil, nil, body), 10, 10, 12, 12)
}

// VirtualHostsPage uses the Windows list-and-editor split. The supplied editor
// owns its platform-specific form, actions and persistence behavior.
func VirtualHostsPage(list, editor fyne.CanvasObject) fyne.CanvasObject {
	left := TitledCard("Domains", "Select a domain to edit it.", nil, list)
	split := container.NewHSplit(left, editor)
	split.Offset = 0.52
	return split
}

// ConfigEditorPage provides the Windows editor hierarchy. Platform adapters own
// file discovery, loading, saving and any extra controls or explanatory status.
// The returned actions belong in the application page header.
func ConfigEditorPage(files fyne.CanvasObject, actions []fyne.CanvasObject, path, pathActions, notice, editor fyne.CanvasObject) (fyne.CanvasObject, fyne.CanvasObject) {
	header := []fyne.CanvasObject{MinSize(files, 240, 0)}
	header = append(header, actions...)
	top := container.NewBorder(nil, nil, nil, pathActions, path)
	heading := []fyne.CanvasObject{top, Gap(1, 4)}
	if notice != nil {
		heading = append(heading, notice)
	}
	body := Surface(container.NewBorder(container.NewVBox(heading...), nil, nil, nil, editor), 10, 10, 12, 12)
	return container.NewHBox(header...), body
}

// Avatar is the Windows service/logo treatment, including its fallback mark.
func Avatar(resource fyne.Resource, name string, size float32) fyne.CanvasObject {
	if resource != nil {
		img := canvas.NewImageFromResource(resource)
		img.FillMode = canvas.ImageFillContain
		img.ScaleMode = canvas.ImageScaleSmooth
		return container.NewGridWrap(fyne.NewSize(size, size), img)
	}
	palette := []uint32{0x7c6cff, 0x22c55e, 0xf59e0b, 0x06b6d4, 0xec4899, 0x3b82f6, 0xef4444, 0x14b8a6}
	sum := 0
	for _, r := range name {
		sum += int(r)
	}
	bg := canvas.NewRectangle(Hex(palette[sum%len(palette)]))
	bg.CornerRadius = size / 4
	letter := "?"
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			letter = strings.ToUpper(string(r))
			break
		}
	}
	text := canvas.NewText(letter, color.White)
	text.TextStyle = fyne.TextStyle{Bold: true}
	text.TextSize = size * 0.5
	text.Alignment = fyne.TextAlignCenter
	return container.NewGridWrap(fyne.NewSize(size, size), container.NewStack(bg, container.NewCenter(text)))
}

// ServiceIdentity places the icon, name, version and badge as in the Windows list.
func ServiceIdentity(avatar, title, version, badge fyne.CanvasObject) fyne.CanvasObject {
	objects := []fyne.CanvasObject{Gap(4, 1), container.NewCenter(avatar), Gap(2, 1), container.NewCenter(title)}
	if version != nil {
		objects = append(objects, container.NewCenter(version))
	}
	if badge != nil {
		objects = append(objects, container.NewCenter(badge))
	}
	return container.NewHBox(objects...)
}

// ServiceRow is the common compact Windows service list layout.
func ServiceRow(identity, controls fyne.CanvasObject) fyne.CanvasObject {
	return Pad(container.NewBorder(nil, nil, identity, controls), 3, 3, 6, 6)
}

// Divider returns the shared one-pixel separator used between compact rows.
func Divider() fyne.CanvasObject {
	line := canvas.NewRectangle(theme.Color(theme.ColorNameSeparator))
	line.SetMinSize(fyne.NewSize(1, 1))
	return line
}

// SectionLabel is the standard uppercase section heading used across pages.
func SectionLabel(value string) fyne.CanvasObject {
	return Pad(MutedText(value, 10.5), 8, 2, 2, 0)
}

// SettingsRow is the Windows settings label and control layout.
func SettingsRow(title, description string, control fyne.CanvasObject) fyne.CanvasObject {
	left := container.NewVBox(BoldText(title, 12.5), MutedText(description, 11.5))
	return Pad(container.NewBorder(nil, nil, left, container.NewCenter(control)), 8, 8, 12, 8)
}

// Sidebar provides the common sidebar frame, divider, width and padding.
// The top and bottom contents remain platform-specific.
func Sidebar(top, bottom fyne.CanvasObject) fyne.CanvasObject {
	body := container.NewBorder(top, bottom, nil, nil)
	background := canvas.NewRectangle(theme.Color(ColorNameSidebar))
	edge := canvas.NewRectangle(theme.Color(theme.ColorNameSeparator))
	return container.NewBorder(
		nil,
		nil,
		nil,
		container.NewGridWrap(fyne.NewSize(1, 1), edge),
		container.NewStack(background, MinSize(Pad(body, 14, 10, 10, 10), 184, 0)),
	)
}

// PageFrame provides the shared header, page inset, footer bar and sidebar
// composition used by both desktop platforms.
func PageFrame(sidebar, title, actions, bottom, pageHost fyne.CanvasObject) fyne.CanvasObject {
	header := container.NewBorder(nil, nil, container.NewCenter(title), actions)
	main := container.NewBorder(
		Pad(header, 12, 8, 18, 18),
		bottom,
		nil,
		nil,
		Pad(pageHost, 0, 10, 18, 18),
	)
	background := canvas.NewRectangle(theme.Color(theme.ColorNameBackground))
	return container.NewBorder(nil, nil, sidebar, nil, container.NewStack(background, main))
}

// Segmented is the shared compact tab selector used for service filters and
// appearance controls on both desktop platforms.
type Segmented struct {
	buttons  []*widget.Button
	selected int
	onChange func(int)
	object   fyne.CanvasObject
}

func NewSegmented(options []string, onChange func(int)) *Segmented {
	segmented := &Segmented{onChange: onChange}
	row := container.NewHBox()
	for i, option := range options {
		index := i
		button := widget.NewButton(option, func() { segmented.SetSelected(index, true) })
		segmented.buttons = append(segmented.buttons, button)
		row.Add(button)
	}
	background := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	background.CornerRadius = 6
	background.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	background.StrokeWidth = 1
	segmented.object = container.NewStack(background, Pad(row, 2, 2, 2, 2))
	if len(options) > 0 {
		segmented.SetSelected(0, false)
	}
	return segmented
}

func (s *Segmented) Object() fyne.CanvasObject { return s.object }

func (s *Segmented) SetSelected(index int, notify bool) {
	if index < 0 || index >= len(s.buttons) {
		return
	}
	s.selected = index
	for i, button := range s.buttons {
		if i == index {
			button.Importance = widget.MediumImportance
		} else {
			button.Importance = widget.LowImportance
		}
		button.Refresh()
	}
	if notify && s.onChange != nil {
		s.onChange(index)
	}
}

// NavItem is the shared interactive sidebar row. Both platforms use the same
// icon size, hit target, radius, selected state and hover treatment.
type NavItem struct {
	widget.BaseWidget
	label    string
	icon     fyne.Resource
	selected bool
	hovered  bool
	onTap    func()

	background *canvas.Rectangle
	image      *widget.Icon
	text       *canvas.Text
}

func NewNavItem(label string, icon fyne.Resource, onTap func()) *NavItem {
	item := &NavItem{label: label, icon: icon, onTap: onTap}
	item.ExtendBaseWidget(item)
	return item
}

func (n *NavItem) SetSelected(selected bool) {
	n.selected = selected
	n.applyState()
}

// RefreshState reapplies the current selection and hover colors after a theme
// change. Call it when the host theme has been replaced.
func (n *NavItem) RefreshState() {
	n.applyState()
}

func (n *NavItem) CreateRenderer() fyne.WidgetRenderer {
	n.background = canvas.NewRectangle(color.Transparent)
	n.background.CornerRadius = 6
	n.image = widget.NewIcon(n.icon)
	n.text = canvas.NewText(n.label, theme.Color(theme.ColorNameForeground))
	n.text.TextSize = 12.5
	row := container.NewHBox(
		container.NewCenter(container.NewGridWrap(fyne.NewSize(15, 15), n.image)),
		n.text,
	)
	content := container.NewStack(n.background, Pad(row, 5, 5, 8, 8))
	n.applyState()
	return widget.NewSimpleRenderer(content)
}

func (n *NavItem) applyState() {
	if n.background == nil || n.text == nil || n.image == nil {
		return
	}
	switch {
	case n.selected:
		n.background.FillColor = theme.Color(ColorNameNavActive)
		n.text.Color = theme.Color(theme.ColorNameForeground)
		n.text.TextStyle = fyne.TextStyle{Bold: true}
		n.image.SetResource(n.icon)
	case n.hovered:
		n.background.FillColor = theme.Color(theme.ColorNameHover)
		n.text.Color = theme.Color(theme.ColorNameForeground)
		n.text.TextStyle = fyne.TextStyle{}
		n.image.SetResource(n.icon)
	default:
		n.background.FillColor = color.Transparent
		n.text.Color = theme.Color(ColorNameMuted)
		n.text.TextStyle = fyne.TextStyle{}
		n.image.SetResource(theme.NewDisabledResource(n.icon))
	}
	n.background.Refresh()
	n.text.Refresh()
}

func (n *NavItem) Tapped(*fyne.PointEvent) {
	if n.onTap != nil {
		n.onTap()
	}
}

func (n *NavItem) MouseIn(*desktop.MouseEvent) {
	n.hovered = true
	n.applyState()
}

func (n *NavItem) MouseMoved(*desktop.MouseEvent) {}

func (n *NavItem) MouseOut() {
	n.hovered = false
	n.applyState()
}

func (n *NavItem) Cursor() desktop.Cursor { return desktop.PointerCursor }

func Hex(value uint32) color.NRGBA {
	return color.NRGBA{R: uint8(value >> 16), G: uint8(value >> 8), B: uint8(value), A: 0xff}
}

func HexA(value uint32, alpha uint8) color.NRGBA {
	result := Hex(value)
	result.A = alpha
	return result
}

// Palette is the single source of truth for GoAMPP's shared desktop colors.
type Palette struct {
	Background, Sidebar, Card, Border, Input, Button, Hover, Pressed color.Color
	Foreground, Muted, Primary, NavActive, Separator                 color.Color
	Menu, Overlay, Header, Scroll, Shadow, Selection, LogBackground  color.Color
	Placeholder, Disabled, DisabledButton                            color.Color
}

func PaletteForVariant(variant fyne.ThemeVariant) Palette {
	if variant == theme.VariantLight {
		return Palette{
			Background: Hex(0xfafafa), Sidebar: Hex(0xf4f4f5), Card: Hex(0xffffff),
			Border: Hex(0xe4e4e7), Input: Hex(0xffffff), Button: Hex(0xf4f4f5),
			Hover: HexA(0x000000, 0x08), Pressed: HexA(0x000000, 0x12),
			Foreground: Hex(0x18181b), Muted: Hex(0x71717a), Primary: Hex(0x2563eb),
			NavActive: HexA(0x000000, 0x08), Separator: Hex(0xe9e9ec),
			Menu: Hex(0xffffff), Overlay: Hex(0xffffff), Header: Hex(0xf4f4f5),
			Scroll: HexA(0x000000, 0x26), Shadow: HexA(0x000000, 0x22),
			Selection: HexA(0x2563eb, 0x30), LogBackground: Hex(0xfafafa),
			Placeholder: Hex(0xa1a1aa), Disabled: Hex(0xa1a1aa), DisabledButton: Hex(0xf4f4f5),
		}
	}
	return Palette{
		Background: Hex(0x0c0c0e), Sidebar: Hex(0x0f0f12), Card: Hex(0x131316),
		Border: Hex(0x222226), Input: Hex(0x18181c), Button: Hex(0x1c1c21),
		Hover: HexA(0xffffff, 0x0c), Pressed: HexA(0xffffff, 0x18),
		Foreground: Hex(0xe4e4e7), Muted: Hex(0x85858f), Primary: Hex(0x4f8cff),
		NavActive: HexA(0xffffff, 0x0e), Separator: Hex(0x1f1f23),
		Menu: Hex(0x17171b), Overlay: Hex(0x131316), Header: Hex(0x131316),
		Scroll: HexA(0xffffff, 0x26), Shadow: HexA(0x000000, 0x66),
		Selection: HexA(0x4f8cff, 0x40), LogBackground: Hex(0x0c0c0e),
		Placeholder: Hex(0x5b5b64), Disabled: Hex(0x55555e), DisabledButton: Hex(0x16161a),
	}
}

type Theme struct {
	forced *fyne.ThemeVariant
}

func NewTheme(preference string) *Theme {
	result := &Theme{}
	switch strings.ToLower(strings.TrimSpace(preference)) {
	case "dark":
		variant := theme.VariantDark
		result.forced = &variant
	case "light":
		variant := theme.VariantLight
		result.forced = &variant
	}
	return result
}

func (t *Theme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if t.forced != nil {
		variant = *t.forced
	}
	p := PaletteForVariant(variant)
	switch name {
	case theme.ColorNameBackground:
		return p.Background
	case theme.ColorNameButton:
		return p.Button
	case theme.ColorNameDisabledButton:
		return p.DisabledButton
	case theme.ColorNameDisabled:
		return p.Disabled
	case theme.ColorNameForeground:
		return p.Foreground
	case theme.ColorNameHover:
		return p.Hover
	case theme.ColorNamePressed:
		return p.Pressed
	case theme.ColorNameInputBackground:
		return p.Input
	case theme.ColorNameInputBorder:
		return p.Border
	case theme.ColorNamePrimary, theme.ColorNameFocus, theme.ColorNameHyperlink:
		return p.Primary
	case theme.ColorNameForegroundOnPrimary, theme.ColorNameForegroundOnError, theme.ColorNameForegroundOnSuccess:
		return color.White
	case theme.ColorNameForegroundOnWarning:
		return Hex(0x1b1200)
	case theme.ColorNameSelection:
		return p.Selection
	case theme.ColorNameSeparator:
		return p.Separator
	case theme.ColorNameMenuBackground:
		return p.Menu
	case theme.ColorNameOverlayBackground:
		return p.Overlay
	case theme.ColorNameHeaderBackground:
		return p.Header
	case theme.ColorNameScrollBar:
		return p.Scroll
	case theme.ColorNameShadow:
		return p.Shadow
	case theme.ColorNamePlaceHolder:
		return p.Placeholder
	case theme.ColorNameSuccess:
		return Hex(0x22c55e)
	case theme.ColorNameWarning:
		return Hex(0xf59e0b)
	case theme.ColorNameError:
		return Hex(0xef4444)
	case ColorNameCard:
		return p.Card
	case ColorNameCardBorder:
		return p.Border
	case ColorNameSidebar:
		return p.Sidebar
	case ColorNameMuted:
		return p.Muted
	case ColorNameNavActive:
		return p.NavActive
	case ColorNameLogBg:
		return p.LogBackground
	default:
		return theme.DefaultTheme().Color(name, variant)
	}
}

func (t *Theme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *Theme) Size(name fyne.ThemeSizeName) float32 {
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
	case theme.SizeNameScrollBarSmall, theme.SizeNameScrollBarRadius:
		return 3
	case theme.SizeNameSeparatorThickness, theme.SizeNameInputBorder:
		return 1
	default:
		return theme.DefaultTheme().Size(name)
	}
}

func (t *Theme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}
