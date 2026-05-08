package components

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font"
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

func (b *Button) Draw(screen *ebiten.Image, font font.Face) {
	displayColor := b.Color
	mx, my := ebiten.CursorPosition()
	isHovered := mx >= b.X && mx <= b.X+b.W && my >= b.Y && my <= b.Y+b.H

	if isHovered && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		displayColor = lighten(b.Color, 0.1)
	}
	vector.DrawFilledRect(screen, float32(b.X), float32(b.Y), float32(b.W), float32(b.H), displayColor, false)
	text.Draw(screen, b.Label, font, b.X+5, b.Y+17, b.LabelColor)
}

func (b *Button) IsClicked(mx, my int) bool {
	return mx >= b.X && mx < b.X+b.W &&
		my >= b.Y && my < b.Y+b.H
}
