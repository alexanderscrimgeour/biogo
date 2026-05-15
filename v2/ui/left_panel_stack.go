package ui

import (
	"biogo/v2/ui/components"

	"github.com/hajimehoshi/ebiten/v2"
)

// stackEntry holds one item in the LeftPanelStack.
type stackEntry struct {
	item    components.Component
	visible bool
}

// LeftPanelStack auto-positions Components vertically on the left side of the
// screen. Panels are registered by index; callers toggle visibility and swap
// items as needed each frame.
type LeftPanelStack struct {
	X       float32
	StartY  float32
	Spacing float32
	entries []stackEntry
}

// Register appends a new slot and returns its index.
// item may be nil to create an initially empty slot.
func (s *LeftPanelStack) Register(item components.Component) int {
	s.entries = append(s.entries, stackEntry{item: item, visible: item != nil})
	return len(s.entries) - 1
}

// Set replaces the item at idx and updates visibility.
func (s *LeftPanelStack) Set(idx int, item components.Component, visible bool) {
	if idx < 0 || idx >= len(s.entries) {
		return
	}
	s.entries[idx].item = item
	s.entries[idx].visible = visible && item != nil
}

// Show marks the entry at idx as visible.
func (s *LeftPanelStack) Show(idx int) {
	if idx >= 0 && idx < len(s.entries) {
		s.entries[idx].visible = s.entries[idx].item != nil
	}
}

// Hide marks the entry at idx as invisible.
func (s *LeftPanelStack) Hide(idx int) {
	if idx >= 0 && idx < len(s.entries) {
		s.entries[idx].visible = false
	}
}

// Draw positions and draws all visible items top-to-bottom.
func (s *LeftPanelStack) Draw(screen *ebiten.Image) {
	y := s.StartY
	for _, e := range s.entries {
		if !e.visible || e.item == nil {
			continue
		}
		_, h := e.item.Draw(screen, s.X, y)
		y += h + s.Spacing
	}
}
