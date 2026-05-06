package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

type point struct {
	X float64
	Y float64
}

type RenderGrid struct {
	position point
	blobSize int
	blobs    []*Blob
	walls    []*Line
}

func NewRenderGrid(xPos, yPos float64, blobSize int) *RenderGrid {
	return &RenderGrid{position: point{X: xPos, Y: yPos}, blobSize: blobSize}
}

func (g *RenderGrid) DrawGrid(image *ebiten.Image) {
	for _, wall := range g.walls {
		wall.Draw(image)
	}
	for _, blob := range g.blobs {
		blob.Draw(image)
	}
}

func (g *RenderGrid) AddLine(minX, minY, maxX, maxY float64) *Line {
	wall := NewLine(minX, minY, maxX, maxY)
	g.walls = append(g.walls, wall)
	return wall
}

func (g *RenderGrid) AddBlob(blobWidth int, c color.Color) *Blob {
	newImage := ebiten.NewImage(blobWidth, blobWidth)
	newImage.Fill(c)
	blob := NewBlob(newImage, &ebiten.GeoM{})
	g.blobs = append(g.blobs, blob)
	return blob
}
