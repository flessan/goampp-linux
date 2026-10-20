//go:build windows

package gui

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

	"goampp/internal/core"
)

// newCard wraps content in a rounded, bordered surface.
func newCard(content fyne.CanvasObject) fyne.CanvasObject {
	bg := canvas.NewRectangle(themeColor(colorCard))
	bg.CornerRadius = theme.Size(theme.SizeNameCardRadius)
	bg.StrokeColor = themeColor(colorCardBorder)
	bg.StrokeWidth = 1
	return container.NewStack(bg, container.New(layout.NewCustomPaddedLayout(10, 10, 12, 12), content))
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
	t := canvas.NewText(s, themeColor(theme.ColorNameForeground))
	t.TextStyle = fyne.TextStyle{Bold: true}
	t.TextSize = size
	return t
}

func mutedText(s string, size float32) *canvas.Text {
	t := canvas.NewText(s, themeColor(colorMuted))
	t.TextSize = size
	return t
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
	r := canvas.NewRectangle(color.Transparent)
	r.SetMinSize(fyne.NewSize(1, h))
	return r
}

func hSpace(w float32) fyne.CanvasObject {
	r := canvas.NewRectangle(color.Transparent)
	r.SetMinSize(fyne.NewSize(w, 1))
	return r
}

// minSized forces obj to at least the given size.
func minSized(obj fyne.CanvasObject, w, h float32) fyne.CanvasObject {
	r := canvas.NewRectangle(color.Transparent)
	r.SetMinSize(fyne.NewSize(w, h))
	return container.NewStack(r, obj)
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

type navItem struct {
	widget.BaseWidget
	label    string
	icon     fyne.Resource
	selected bool
	hovered  bool
	onTap    func()

	bg   *canvas.Rectangle
	img  *widget.Icon
	text *canvas.Text
}

func newNavItem(label string, icon fyne.Resource, onTap func()) *navItem {
	n := &navItem{label: label, icon: icon, onTap: onTap}
	n.ExtendBaseWidget(n)
	return n
}

func (n *navItem) CreateRenderer() fyne.WidgetRenderer {
	n.bg = canvas.NewRectangle(color.Transparent)
	n.bg.CornerRadius = 6
	n.img = widget.NewIcon(n.icon)
	n.text = canvas.NewText(n.label, themeColor(theme.ColorNameForeground))
	n.text.TextSize = 12.5
	row := container.NewHBox(container.NewCenter(container.NewGridWrap(fyne.NewSize(15, 15), n.img)), n.text)
	content := container.NewStack(n.bg, container.New(layout.NewCustomPaddedLayout(5, 5, 8, 8), row))
	n.applyState()
	return widget.NewSimpleRenderer(content)
}

func (n *navItem) setSelected(v bool) {
	n.selected = v
	n.applyState()
}

func (n *navItem) applyState() {
	if n.bg == nil {
		return
	}
	switch {
	case n.selected:
		n.bg.FillColor = themeColor(colorNavActive)
		n.text.Color = themeColor(theme.ColorNameForeground)
		n.text.TextStyle = fyne.TextStyle{Bold: true}
		n.img.SetResource(n.icon)
	case n.hovered:
		n.bg.FillColor = themeColor(theme.ColorNameHover)
		n.text.Color = themeColor(theme.ColorNameForeground)
		n.text.TextStyle = fyne.TextStyle{}
		n.img.SetResource(n.icon)
	default:
		n.bg.FillColor = color.Transparent
		n.text.Color = themeColor(colorMuted)
		n.text.TextStyle = fyne.TextStyle{}
		n.img.SetResource(theme.NewDisabledResource(n.icon))
	}
	n.bg.Refresh()
	n.text.Refresh()
}

func (n *navItem) Tapped(*fyne.PointEvent) {
	if n.onTap != nil {
		n.onTap()
	}
}

func (n *navItem) MouseIn(*desktop.MouseEvent)    { n.hovered = true; n.applyState() }
func (n *navItem) MouseMoved(*desktop.MouseEvent) {}
func (n *navItem) MouseOut()                      { n.hovered = false; n.applyState() }
func (n *navItem) Cursor() desktop.Cursor         { return desktop.PointerCursor }

// ---- segmented control ----

type segmented struct {
	buttons  []*widget.Button
	selected int
	onChange func(int)
	obj      fyne.CanvasObject
}

func newSegmented(options []string, onChange func(int)) *segmented {
	s := &segmented{onChange: onChange}
	row := container.NewHBox()
	for i, opt := range options {
		i := i
		b := widget.NewButton(opt, func() { s.setSelected(i, true) })
		s.buttons = append(s.buttons, b)
		row.Add(b)
	}
	bg := canvas.NewRectangle(themeColor(theme.ColorNameInputBackground))
	bg.CornerRadius = 7
	s.obj = container.NewStack(bg, container.New(layout.NewCustomPaddedLayout(2, 2, 2, 2), row))
	s.setSelected(0, false)
	return s
}

func (s *segmented) setSelected(i int, notify bool) {
	s.selected = i
	for j, b := range s.buttons {
		if j == i {
			b.Importance = widget.MediumImportance
		} else {
			b.Importance = widget.LowImportance
		}
		b.Refresh()
	}
	if notify && s.onChange != nil {
		s.onChange(i)
	}
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
