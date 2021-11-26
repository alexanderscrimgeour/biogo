package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

type point struct {
	X float64
	Y float64
}

type Grid struct {
	position point
	blobSize int

	blobs []*Blob

	walls []*Line
}

func NewGrid(xPos, yPos float64, blobSize int) *Grid {
	grid := &Grid{position: point{X: xPos, Y: yPos}, blobSize: blobSize}
	return grid
}

func (g *Grid) DrawGrid(image *ebiten.Image) {
	for _, wall := range g.walls {
		wall.Draw(image)
	}
	for _, blob := range g.blobs {
		blob.Draw(image)
	}
}

func (g *Grid) AddLine(minX, minY, maxX, maxY float64) *Line {
	wall := NewLine(minX, minY, maxX, maxY)
	g.walls = append(g.walls, wall)
	return wall
}

func (g *Grid) AddBlob(blobWidth int, c color.Color) *Blob {
	var newImage *ebiten.Image
	newImage = ebiten.NewImage(blobWidth, blobWidth)
	newImage.Fill(c)
	blob := NewBlob(newImage, &ebiten.GeoM{})
	g.blobs = append(g.blobs, blob)
	return blob
}
