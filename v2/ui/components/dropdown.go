package components

import (
	"image/color"

	"biogo/v2/ui/colors"

	"github.com/hajimehoshi/ebiten/v2"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	menuBarH       = float32(34)
	menuBarPad     = float32(10)
	menuBarSpacing = float32(8)

	ddPad       = float32(8)
	ddTitleH    = float32(20)
	ddSliderGap = float32(6)
)

type DropdownItem struct {
	H        float32
	Draw     func(screen *ebiten.Image, x, y float32)
	OnDown   func(mx, my int)
	OnDrag   func(mx int)
	Release  func()
	Dragging func() bool
}

// Dropdown is a panel anchored below a menu-bar button containing sliders and range sliders.
type Dropdown struct {
	open       bool
	trigger    *Button
	title      string
	titleColor color.Color
	panelW     float32
	font       *textv2.GoXFace
	Items      []DropdownItem
}

func NewDropdown(font *textv2.GoXFace, trigger *Button, title string, titleColor color.Color, panelW float32) *Dropdown {
	return &Dropdown{
		trigger:    trigger,
		title:      title,
		titleColor: titleColor,
		panelW:     panelW,
		font:       font,
	}
}

func (d *Dropdown) AddSlider(s *Slider) {
	d.Items = append(d.Items, DropdownItem{
		H:    s.H,
		Draw: func(screen *ebiten.Image, x, y float32) { s.Draw(screen, x, y) },
		OnDown: func(mx, my int) {
			if s.InBounds(mx, my) {
				s.Dragging = true
				s.UpdateValue(mx)
			}
		},
		OnDrag: func(mx int) {
			if s.Dragging {
				s.UpdateValue(mx)
			}
		},
		Release:  func() { s.Dragging = false },
		Dragging: func() bool { return s.Dragging },
	})
}

func (d *Dropdown) AddButton(b *Button) {
	d.Items = append(d.Items, DropdownItem{
		H:    b.H,
		Draw: func(screen *ebiten.Image, x, y float32) { b.Draw(screen, x, y) },
		OnDown: func(mx, my int) {
			if b.IsClicked(mx, my) && b.OnClick != nil {
				b.OnClick()
			}
		},
		OnDrag:   func(mx int) {},
		Release:  func() {},
		Dragging: func() bool { return false },
	})
}

func (d *Dropdown) AddProportionSliders(ps *ProportionSliders) {
	d.Items = append(d.Items, DropdownItem{
		H:        ps.H,
		Draw:     func(screen *ebiten.Image, x, y float32) { ps.Draw(screen, x, y) },
		OnDown:   ps.HandleDown,
		OnDrag:   ps.HandleDrag,
		Release:  ps.Release,
		Dragging: ps.IsDragging,
	})
}

func (d *Dropdown) AddFountainPanel(fp *FountainPanel) {
	d.Items = append(d.Items, DropdownItem{
		H:        fp.H,
		Draw:     func(screen *ebiten.Image, x, y float32) { fp.Draw(screen, x, y) },
		OnDown:   fp.HandleDown,
		OnDrag:   fp.HandleDrag,
		Release:  fp.Release,
		Dragging: fp.IsDragging,
	})
}

func (d *Dropdown) AddRangeSlider(s *RangeSlider) {
	d.Items = append(d.Items, DropdownItem{
		H:        s.H,
		Draw:     func(screen *ebiten.Image, x, y float32) { s.Draw(screen, x, y) },
		OnDown:   func(mx, my int) { s.HandleMouseDown(mx, my) },
		OnDrag:   func(mx int) { s.HandleDrag(mx) },
		Release:  s.Release,
		Dragging: s.IsDragging,
	})
}

func (d *Dropdown) Toggle()      { d.open = !d.open }
func (d *Dropdown) Close()       { d.open = false }
func (d *Dropdown) IsOpen() bool { return d.open }

func (d *Dropdown) AnyDragging() bool {
	for _, item := range d.Items {
		if item.Dragging() {
			return true
		}
	}
	return false
}

func (d *Dropdown) panelBounds() (x, y, w, h float32) {
	bx, _, _, _ := d.trigger.Bounds()
	ph := ddPad*2 + ddTitleH
	for _, item := range d.Items {
		ph += ddSliderGap + item.H
	}
	return bx, menuBarH, d.panelW, ph
}

// HandleClick routes a mouse-down into the panel. Returns true if consumed.
func (d *Dropdown) HandleClick(mx, my int) bool {
	if !d.open {
		return false
	}
	px, py, pw, ph := d.panelBounds()
	if float32(mx) < px || float32(mx) >= px+pw || float32(my) < py || float32(my) >= py+ph {
		return false
	}
	for _, item := range d.Items {
		item.OnDown(mx, my)
	}
	return true
}

func (d *Dropdown) HandleDrag(mx int) {
	if !d.open {
		return
	}
	for _, item := range d.Items {
		item.OnDrag(mx)
	}
}

func (d *Dropdown) HandleRelease() {
	for _, item := range d.Items {
		item.Release()
	}
}

func (d *Dropdown) Draw(screen *ebiten.Image) {
	if !d.open {
		return
	}
	px, py, pw, ph := d.panelBounds()
	vector.FillRect(screen, px, py, pw, ph, colors.ColorDropdownBG, false)
	vector.StrokeRect(screen, px, py, pw, ph, 1, colors.ColorDropdownBorder, false)

	if d.font != nil {
		m := d.font.Metrics()
		textH := m.HLineGap + m.HAscent + m.HDescent
		ty := float64(py+ddPad) + (float64(ddTitleH)-textH)/2
		op := &textv2.DrawOptions{}
		op.GeoM.Translate(float64(px+ddPad), ty)
		tc := d.titleColor
		if tc == nil {
			tc = colors.ColorLabelPrimary
		}
		op.ColorScale.ScaleWithColor(tc)
		textv2.Draw(screen, d.title, d.font, op)
	}

	sy := py + ddPad + ddTitleH
	for _, item := range d.Items {
		sy += ddSliderGap
		item.Draw(screen, px+ddPad, sy)
		sy += item.H
	}
}
