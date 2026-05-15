package ui

import (
	"biogo/v2/ui/components"

	"github.com/hajimehoshi/ebiten/v2"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
)

type UserInterface struct {
	font       *textv2.GoXFace
	pauseBtn   *components.Button
	fovBtn     *components.Button
	newGameBtn *components.Button
	themeBtn   *components.Button
}

func NewUI(f *textv2.GoXFace) *UserInterface {
	return &UserInterface{}
}

func (ui *UserInterface) Draw(screen *ebiten.Image) {
	ui.pauseBtn.Draw(screen, ui.font)
	ui.fovBtn.Draw(screen, ui.font)
	ui.newGameBtn.Draw(screen, ui.font)
	ui.themeBtn.Draw(screen, ui.font)
	// ui.mutSlider.Draw(screen, ui.font)
	// ui.history.Draw(screen, ui.font)
}
