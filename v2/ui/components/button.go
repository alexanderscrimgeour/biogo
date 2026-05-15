package components

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var (
	ColorDefault       = color.RGBA{120, 144, 156, 100}
	ColorButtonPressed = color.RGBA{144, 164, 174, 100}
	ColorButtonGreen   = color.RGBA{76, 175, 80, 100}
	ColorButtonRed     = color.RGBA{244, 67, 54, 100}
)

type Button struct {
	X, Y, W, H int
	Label      string
	LabelColor color.Gray16
	Color      color.RGBA
	OnClick    func()
}

func (b *Button) Draw(screen *ebiten.Image, font *textv2.GoXFace) {
	displayColor := b.Color
	mx, my := ebiten.CursorPosition()
	isHovered := mx >= b.X && mx <= b.X+b.W && my >= b.Y && my <= b.Y+b.H

	if isHovered && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		displayColor = lighten(b.Color, 0.1)
	}
	vector.FillRect(screen, float32(b.X), float32(b.Y), float32(b.W), float32(b.H), displayColor, false)
	op := &textv2.DrawOptions{}
	op.GeoM.Translate(float64(b.X+5), float64(b.Y+17))
	op.ColorScale.ScaleWithColor(b.LabelColor)
	textv2.Draw(screen, b.Label, font, op)
}

func (b *Button) IsClicked(mx, my int) bool {
	return mx >= b.X && mx < b.X+b.W &&
		my >= b.Y && my < b.Y+b.H
}
