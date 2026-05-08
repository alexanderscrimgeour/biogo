package ui

import (
	"biogo/v2/ui/components"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font"
)

type UserInterface struct {
	font       font.Face
	pauseBtn   *components.Button
	fovBtn     *components.Button
	newGameBtn *components.Button
	themeBtn   *components.Button
	mutSlider  *components.Slider
	history    *components.Graph
}

func NewUI(f font.Face) *UserInterface {
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
