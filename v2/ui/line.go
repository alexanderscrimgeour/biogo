package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/colornames"
)

type Line struct {
	img  *ebiten.Image
	geoM *ebiten.GeoM
}

func NewLine(minX, minY, maxX, maxY float64) *Line {
	width := int(maxX - minX)
	height := int(maxY - minY)
	img := ebiten.NewImage(width, height)
	img.Fill(colornames.White)
	geoM := &ebiten.GeoM{}
	geoM.Translate(minX, minY)
	return &Line{img: img, geoM: geoM}
}

func (l *Line) Draw(targetImage *ebiten.Image) {
	targetImage.DrawImage(l.img, &ebiten.DrawImageOptions{GeoM: *l.geoM})
}
