package ui

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

type blob struct {
	length     int8
	width      int8
	focalPoint point
}

type Blob struct {
	img    *ebiten.Image
	geoM   *ebiten.GeoM
	center *point
}

func NewBlob(img *ebiten.Image, geoM *ebiten.GeoM) *Blob {
	point := &point{X: float64(img.Bounds().Dx() / 2), Y: float64(img.Bounds().Dy() / 2)}
	return &Blob{img: img, geoM: geoM, center: point}
}

func (b *Blob) GetImage() *ebiten.Image {
	return b.img
}

func (b *Blob) GetImageOptions() *ebiten.DrawImageOptions {
	return &ebiten.DrawImageOptions{GeoM: *b.geoM}
}

func (b *Blob) GetSize() (width, height int) {
	return b.img.Size()
}

func (b *Blob) GetHeight() (height int) {
	_, height = b.img.Size()
	return
}

func (b *Blob) GetWidth() (width int) {
	width, _ = b.img.Size()
	return
}

func (b *Blob) GetCenter() *point {
	return b.center
}

func (b *Blob) Translate(x, y float64) {
	b.geoM.Translate(x, y)
	b.updateCenter(x, y)
}

func (b *Blob) Move(x, y float64) {

	moveX := x - b.center.X
	moveY := y - b.center.Y
	b.geoM.Translate(moveX, moveY)
	b.SetCenter(x, y)
}

func (b *Blob) Draw(targetImage *ebiten.Image) {
	targetImage.DrawImage(b.img, b.GetImageOptions())
}

func (b *Blob) Rotate(angle float64, focalPoint point) {
	dX := focalPoint.X - b.center.X
	dY := focalPoint.Y - b.center.Y

	rotatedX := math.Cos(-angle)*dX - math.Sin(-angle)*dY + focalPoint.X
	rotatedY := math.Sin(-angle)*dX + math.Cos(-angle)*dY + focalPoint.Y

	dX = rotatedX - b.center.X
	dY = rotatedY - b.center.Y

	b.Translate(dX, dY)
}

func (b *Blob) SetCenter(x, y float64) {
	b.center.X = x
	b.center.Y = y
}

func (b *Blob) updateCenter(x, y float64) {
	b.center.X += x
	b.center.Y += y
}
