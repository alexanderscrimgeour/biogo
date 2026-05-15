package components

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Component is implemented by every renderable UI element.
// Draw renders at (x, y) and returns (width, height) consumed.
// Size returns preferred (width, height) for layout without rendering.
type Component interface {
	Draw(screen *ebiten.Image, x, y float32) (float32, float32)
	Size() (float32, float32)
}

type panelRow struct {
	items []Component
}

// Panel is a container that stacks rows of components vertically.
// Its height grows automatically to fit content.
// Rows added via AddRow lay components side by side, sharing width equally.
type Panel struct {
	W         float32
	Padding   float32
	Spacing   float32
	BaseColor color.RGBA
	Border    color.RGBA
	rows      []panelRow
	lastX     float32
	lastY     float32
}

// Add appends a single full-width component as a new row.
func (p *Panel) Add(c Component) *Panel {
	p.rows = append(p.rows, panelRow{items: []Component{c}})
	return p
}

// AddRow appends multiple components side by side as one row.
func (p *Panel) AddRow(cs ...Component) *Panel {
	p.rows = append(p.rows, panelRow{items: cs})
	return p
}

// Height returns the computed total height based on current row contents.
func (p *Panel) Height() float32 {
	h := p.Padding
	for i, row := range p.rows {
		if i > 0 {
			h += p.Spacing
		}
		rowH := float32(0)
		for _, c := range row.items {
			_, ch := c.Size()
			if ch > rowH {
				rowH = ch
			}
		}
		h += rowH
	}
	return h + p.Padding
}

// perItemW returns available width per item within a row.
func (p *Panel) perItemW(n int) float32 {
	avail := p.W - p.Padding*2
	if n > 1 {
		avail -= p.Spacing * float32(n-1)
	}
	if n > 0 {
		return avail / float32(n)
	}
	return avail
}

// Draw renders the panel background and all child components at (x, y).
// Implements Component so panels can be nested.
func (p *Panel) Draw(screen *ebiten.Image, x, y float32) (float32, float32) {
	p.lastX, p.lastY = x, y
	h := p.Height()
	vector.FillRect(screen, x, y, p.W, h, p.BaseColor, false)
	if p.Border.A > 0 {
		vector.StrokeRect(screen, x, y, p.W, h, 1, p.Border, false)
	}
	currY := y + p.Padding
	for i, row := range p.rows {
		if i > 0 {
			currY += p.Spacing
		}
		currX := x + p.Padding
		itemW := p.perItemW(len(row.items))
		rowH := float32(0)
		for j, c := range row.items {
			if j > 0 {
				currX += p.Spacing
			}
			_, ch := c.Draw(screen, currX, currY)
			if ch > rowH {
				rowH = ch
			}
			currX += itemW
		}
		currY += rowH
	}
	return p.W, h
}

// Size returns (W, Height()) for use as a Component child.
func (p *Panel) Size() (float32, float32) {
	return p.W, p.Height()
}

// Contains reports whether (x, y) falls inside the last drawn area.
func (p *Panel) Contains(x, y float32) bool {
	h := p.Height()
	return x >= p.lastX && x <= p.lastX+p.W && y >= p.lastY && y <= p.lastY+h
}
