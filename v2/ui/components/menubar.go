package components

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// menuBarItem wraps a Component with input callbacks for use in a MenuBar.
type menuBarItem struct {
	comp      Component
	onClick   func(mx, my int) bool
	onDrag    func(mx int)
	onRelease func()
}

// MenuBar lays buttons and sliders horizontally across the top of the screen.
// Items are auto-positioned left-to-right with Spacing between them.
// Right-aligned items are positioned flush to the right edge.
type MenuBar struct {
	H          float32
	Padding    float32
	Spacing    float32
	Color      color.RGBA
	items      []menuBarItem
	rightItems []menuBarItem
}

// AddButton appends a button. The button's OnClick is called on click.
func (mb *MenuBar) AddButton(b *Button) *MenuBar {
	mb.items = append(mb.items, menuBarItem{
		comp: b,
		onClick: func(mx, my int) bool {
			if b.IsClicked(mx, my) {
				if b.OnClick != nil {
					b.OnClick()
				}
				return true
			}
			return false
		},
		onDrag:    func(mx int) {},
		onRelease: func() {},
	})
	return mb
}

// AddButtonRight appends a button aligned to the right edge of the menu bar.
func (mb *MenuBar) AddButtonRight(b *Button) *MenuBar {
	mb.rightItems = append(mb.rightItems, menuBarItem{
		comp: b,
		onClick: func(mx, my int) bool {
			if b.IsClicked(mx, my) {
				if b.OnClick != nil {
					b.OnClick()
				}
				return true
			}
			return false
		},
		onDrag:    func(mx int) {},
		onRelease: func() {},
	})
	return mb
}

// AddSliderRight appends a slider aligned to the right edge of the menu bar.
func (mb *MenuBar) AddSliderRight(s *Slider) *MenuBar {
	mb.rightItems = append(mb.rightItems, menuBarItem{
		comp: s,
		onClick: func(mx, my int) bool {
			if s.InBounds(mx, my) {
				s.Dragging = true
				s.UpdateValue(mx)
				return true
			}
			return false
		},
		onDrag: func(mx int) {
			if s.Dragging {
				s.UpdateValue(mx)
			}
		},
		onRelease: func() { s.Dragging = false },
	})
	return mb
}

// AddSlider appends a slider. Drag state is managed by HandleDrag/HandleRelease.
func (mb *MenuBar) AddSlider(s *Slider) *MenuBar {
	mb.items = append(mb.items, menuBarItem{
		comp: s,
		onClick: func(mx, my int) bool {
			if s.InBounds(mx, my) {
				s.Dragging = true
				s.UpdateValue(mx)
				return true
			}
			return false
		},
		onDrag: func(mx int) {
			if s.Dragging {
				s.UpdateValue(mx)
			}
		},
		onRelease: func() {
			s.Dragging = false
		},
	})
	return mb
}

// Draw renders the menu bar background and all items.
func (mb *MenuBar) Draw(screen *ebiten.Image) {
	sw := float32(screen.Bounds().Dx())
	vector.FillRect(screen, 0, 0, sw, mb.H, mb.Color, false)
	itemY := (mb.H - 24) / 2
	x := mb.Padding
	for _, item := range mb.items {
		w, _ := item.comp.Size()
		item.comp.Draw(screen, x, itemY)
		x += w + mb.Spacing
	}
	// Right-aligned items: lay out from right edge inward (last added = rightmost).
	rx := sw - mb.Padding
	for i := len(mb.rightItems) - 1; i >= 0; i-- {
		item := mb.rightItems[i]
		w, _ := item.comp.Size()
		rx -= w
		item.comp.Draw(screen, rx, itemY)
		rx -= mb.Spacing
	}
}

// HandleClick processes a mouse-down event; returns true if consumed.
func (mb *MenuBar) HandleClick(mx, my int) bool {
	for _, item := range mb.items {
		if item.onClick(mx, my) {
			return true
		}
	}
	for _, item := range mb.rightItems {
		if item.onClick(mx, my) {
			return true
		}
	}
	return false
}

// HandleDrag forwards continuous mouse movement to draggable items.
func (mb *MenuBar) HandleDrag(mx int) {
	for _, item := range mb.items {
		item.onDrag(mx)
	}
	for _, item := range mb.rightItems {
		item.onDrag(mx)
	}
}

// HandleRelease ends any active drags.
func (mb *MenuBar) HandleRelease() {
	for _, item := range mb.items {
		item.onRelease()
	}
	for _, item := range mb.rightItems {
		item.onRelease()
	}
}
