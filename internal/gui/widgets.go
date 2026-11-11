//go:build windows

package gui

import (
	"image/color"
	"strings"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"goampp/internal/core"
	"goampp/internal/uikit"
)

// newCard wraps content in a rounded, bordered surface.
func newCard(content fyne.CanvasObject) fyne.CanvasObject {
	return uikit.Surface(content, 10, 10, 12, 12)
}

// newTitledCard is a card with a small heading and optional actions on the right.
func newTitledCard(title, subtitle string, actions fyne.CanvasObject, body fyne.CanvasObject) fyne.CanvasObject {
	head := container.NewVBox(boldText(title, 13))
	if subtitle != "" {
		head.Add(mutedText(subtitle, 11.5))
	}
	var top fyne.CanvasObject = head
	if actions != nil {
		top = container.NewBorder(nil, nil, nil, container.NewCenter(actions), head)
	}
	if body == nil {
		return newCard(top)
	}
	return newCard(container.NewBorder(container.NewVBox(top, vSpace(4)), nil, nil, nil, body))
}

func boldText(s string, size float32) *canvas.Text {
	return uikit.BoldText(s, size)
}

func mutedText(s string, size float32) *canvas.Text {
	return uikit.MutedText(s, size)
}

// wrapMuted is a muted label that wraps, for descriptions.
func wrapMuted(s string) *widget.RichText {
	rt := widget.NewRichText(&widget.TextSegment{
		Text:  s,
		Style: widget.RichTextStyle{ColorName: colorMuted, Inline: true, SizeName: theme.SizeNameCaptionText},
	})
	rt.Wrapping = fyne.TextWrapWord
	return rt
}

func truncLabel(s string) *widget.Label {
	l := widget.NewLabel(s)
	l.Truncation = fyne.TextTruncateEllipsis
	return l
}

func vSpace(h float32) fyne.CanvasObject {
	return uikit.Gap(1, h)
}

func hSpace(w float32) fyne.CanvasObject {
	return uikit.Gap(w, 1)
}

// minSized forces obj to at least the given size.
func minSized(obj fyne.CanvasObject, w, h float32) fyne.CanvasObject {
	return uikit.MinSize(obj, w, h)
}

func lowButton(label string, icon fyne.Resource, tapped func()) *widget.Button {
	b := widget.NewButtonWithIcon(label, icon, tapped)
	b.Importance = widget.LowImportance
	return b
}

func importantButton(label string, icon fyne.Resource, imp widget.Importance, tapped func()) *widget.Button {
	b := widget.NewButtonWithIcon(label, icon, tapped)
	b.Importance = imp
	return b
}

// ---- status pill ----

type statusPill struct {
	bg   *canvas.Rectangle
	dot  *canvas.Circle
	text *canvas.Text
	obj  fyne.CanvasObject
}

func newStatusPill() *statusPill {
	p := &statusPill{
		bg:   canvas.NewRectangle(color.Transparent),
		dot:  canvas.NewCircle(color.Transparent),
		text: canvas.NewText("", color.Transparent),
	}
	p.text.TextSize = 11.5
	dot := container.NewGridWrap(fyne.NewSize(6, 6), p.dot)
	p.obj = container.NewHBox(container.NewCenter(dot), p.text)
	return p
}

func (p *statusPill) set(text string, c color.Color) {
	p.dot.FillColor = c
	p.text.Color = themeColor(colorMuted)
	p.text.Text = text
	p.dot.Refresh()
	p.text.Refresh()
}

// ---- letter avatar for services without a logo ----

var avatarColors = []uint32{0x7c6cff, 0x22c55e, 0xf59e0b, 0x06b6d4, 0xec4899, 0x3b82f6, 0xef4444, 0x14b8a6}

func letterAvatar(name string, size float32) fyne.CanvasObject {
	sum := 0
	for _, r := range name {
		sum += int(r)
	}
	bg := canvas.NewRectangle(hex(avatarColors[sum%len(avatarColors)]))
	bg.CornerRadius = size / 4
	letter := "?"
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			letter = strings.ToUpper(string(r))
			break
		}
	}
	t := canvas.NewText(letter, color.White)
	t.TextStyle = fyne.TextStyle{Bold: true}
	t.TextSize = size * 0.5
	t.Alignment = fyne.TextAlignCenter
	return container.NewGridWrap(fyne.NewSize(size, size), container.NewStack(bg, container.NewCenter(t)))
}

func serviceAvatar(name string, size float32) fyne.CanvasObject {
	res := serviceIcon(name)
	if res == nil {
		return letterAvatar(name, size)
	}
	img := canvas.NewImageFromResource(res)
	img.FillMode = canvas.ImageFillContain
	img.ScaleMode = canvas.ImageScaleSmooth
	return container.NewGridWrap(fyne.NewSize(size, size), img)
}

// ---- sidebar navigation item ----

type navItem = uikit.NavItem

func newNavItem(label string, icon fyne.Resource, onTap func()) *navItem {
	return uikit.NewNavItem(label, icon, onTap)
}

// ---- segmented control ----

type segmented struct {
	shared *uikit.Segmented
	obj    fyne.CanvasObject
}

func newSegmented(options []string, onChange func(int)) *segmented {
	control := uikit.NewSegmented(options, onChange)
	return &segmented{shared: control, obj: control.Object()}
}

func (s *segmented) setSelected(index int, notify bool) {
	s.shared.SetSelected(index, notify)
}

// ---- tappable surface that reports right clicks ----

type contextSurface struct {
	widget.BaseWidget
	content     fyne.CanvasObject
	onSecondary func(pos fyne.Position)
}

func newContextSurface(content fyne.CanvasObject, onSecondary func(fyne.Position)) *contextSurface {
	c := &contextSurface{content: content, onSecondary: onSecondary}
	c.ExtendBaseWidget(c)
	return c
}

func (c *contextSurface) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(c.content)
}

func (c *contextSurface) TappedSecondary(e *fyne.PointEvent) {
	if c.onSecondary != nil {
		c.onSecondary(e.AbsolutePosition)
	}
}

// showMenuAt pops up a menu below obj.
func showMenuBelow(menu *fyne.Menu, obj fyne.CanvasObject) {
	if core.App.Win == nil {
		return
	}
	widget.ShowPopUpMenuAtRelativePosition(menu, core.App.Win.Canvas(), fyne.NewPos(0, obj.Size().Height), obj)
}
