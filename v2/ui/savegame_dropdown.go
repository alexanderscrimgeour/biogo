package ui

import (
	"biogo/v2/simulation"
	"biogo/v2/ui/components"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	saveGamePanelW = float32(285)
	saveGameRowH   = float32(28)
)

// newSaveGameDropdown creates the Save Game dropdown panel (no items yet; call
// rebuildSaveGameItems before each open to refresh the save list).
func newSaveGameDropdown(font *textv2.GoXFace, trigger *components.Button) *Dropdown {
	return newDropdown(font, trigger, "Save Game", ColorLabelPrimary, saveGamePanelW)
}

// rebuildSaveGameItems repopulates dd.items with:
//  1. A text-input row for the new-save name.
//  2. A "Save as New" button.
//  3. A separator + one row per existing save (Load / Overwrite buttons).
//
// The TextInputField nameInput is owned by UserInterface so its Text/Focused
// fields can be updated each frame without rebuilding the item list.
func rebuildSaveGameItems(
	dd *Dropdown,
	font *textv2.GoXFace,
	nameInput *components.TextInputField,
	getName func() string,
	getFocused func() bool,
	onFocusInput func(),
	onSaveNew func(),
	saves []simulation.SavedGame,
	onLoad func(path string),
	onOverwrite func(path string),
) {
	dd.items = dd.items[:0]

	// ── Text input ────────────────────────────────────────────────────────────
	dd.items = append(dd.items, ddItem{
		h: nameInput.H,
		draw: func(screen *ebiten.Image, x, y float32) {
			nameInput.W = saveGamePanelW - ddPad*2
			nameInput.Text = getName()
			nameInput.Focused = getFocused()
			nameInput.Draw(screen, x, y)
		},
		onDown: func(mx, my int) {
			if nameInput.Contains(float32(mx), float32(my)) {
				onFocusInput()
			}
		},
		onDrag:   func(_ int) {},
		release:  func() {},
		dragging: func() bool { return false },
	})

	// ── Save as New button ────────────────────────────────────────────────────
	saveNewBtn := &components.Button{
		W:          saveGamePanelW - ddPad*2,
		H:          24,
		Label:      "Save as New",
		Color:      ColorBtnSave,
		LabelColor: color.White,
		Font:       font,
	}
	saveNewBtn.OnClick = onSaveNew
	dd.addButton(saveNewBtn)

	if len(saves) == 0 {
		return
	}

	// ── Separator ─────────────────────────────────────────────────────────────
	dd.items = append(dd.items, ddItem{
		h: 22,
		draw: func(screen *ebiten.Image, x, y float32) {
			mid := y + 11
			lineEnd := x + saveGamePanelW - ddPad*2
			vector.StrokeLine(screen, x, mid, lineEnd, mid, 1, ColorDropdownBorder, false)
			if font != nil {
				op := &textv2.DrawOptions{}
				op.GeoM.Translate(float64(x+4), float64(mid-8))
				op.ColorScale.ScaleWithColor(ColorLabelPrimary)
				textv2.Draw(screen, "Saved Games", font, op)
			}
		},
		onDown:   func(_ int, _ int) {},
		onDrag:   func(_ int) {},
		release:  func() {},
		dragging: func() bool { return false },
	})

	// ── Per-save rows ─────────────────────────────────────────────────────────
	contentW := saveGamePanelW - ddPad*2
	const loadW = float32(50)
	const overW = float32(78)
	const btnGap = float32(3)
	nameLabelW := contentW - loadW - overW - btnGap*2

	for _, save := range saves {
		save := save // capture

		loadBtn := &components.Button{
			W: loadW, H: saveGameRowH,
			Label:      "Load",
			Color:      ColorBtnSave,
			LabelColor: color.White,
			Font:       font,
		}
		loadBtn.OnClick = func() { onLoad(save.Path) }

		overBtn := &components.Button{
			W: overW, H: saveGameRowH,
			Label:      "Overwrite",
			Color:      ColorBtnEdit,
			LabelColor: color.White,
			Font:       font,
		}
		overBtn.OnClick = func() { onOverwrite(save.Path) }

		saveName := save.Name
		dd.items = append(dd.items, ddItem{
			h: saveGameRowH,
			draw: func(screen *ebiten.Image, x, y float32) {
				if font != nil {
					display := sgTruncate(saveName, nameLabelW, font)
					op := &textv2.DrawOptions{}
					op.GeoM.Translate(float64(x+2), float64(y+6))
					op.ColorScale.ScaleWithColor(color.White)
					textv2.Draw(screen, display, font, op)
				}
				loadBtn.Draw(screen, x+nameLabelW+btnGap, y)
				overBtn.Draw(screen, x+nameLabelW+btnGap+loadW+btnGap, y)
			},
			onDown: func(mx, my int) {
				if loadBtn.IsClicked(mx, my) && loadBtn.OnClick != nil {
					loadBtn.OnClick()
				}
				if overBtn.IsClicked(mx, my) && overBtn.OnClick != nil {
					overBtn.OnClick()
				}
			},
			onDrag:   func(_ int) {},
			release:  func() {},
			dragging: func() bool { return false },
		})
	}
}

// sgTruncate trims s to fit within maxW pixels, appending "…" if needed.
func sgTruncate(s string, maxW float32, face *textv2.GoXFace) string {
	if face == nil || len(s) == 0 {
		return s
	}
	runes := []rune(s)
	for {
		w, _ := textv2.Measure(string(runes), face, 0)
		if float32(w) <= maxW || len(runes) == 0 {
			return string(runes)
		}
		runes = runes[:len(runes)-1]
	}
}
