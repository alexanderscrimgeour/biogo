package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/colornames"
)

type StatLine struct {
	point       *point
	img         *ebiten.Image
	description string
	value       string
	color       color.Color
}

func NewStatLine(img *ebiten.Image, description, value string) *StatLine {
	p := &point{X: float64(img.Bounds().Dx() / 2), Y: float64(img.Bounds().Dy() / 2)}
	return &StatLine{
		point:       p,
		img:         img,
		description: description,
		value:       value,
		color:       colornames.Black,
	}
}

func (s *StatLine) DrawStatLine(image *ebiten.Image) {

}
