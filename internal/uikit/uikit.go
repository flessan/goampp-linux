// Package uikit contains the shared desktop UI primitives used by the
// Windows and Linux front ends. Platform-specific behavior belongs in the
// platform front ends; visual structure and spacing live here.
package uikit

import (
	"image/color"

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

// SectionLabel is the standard uppercase section heading used across pages.
func SectionLabel(value string) fyne.CanvasObject {
	return Pad(MutedText(value, 10.5), 8, 2, 2, 0)
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
