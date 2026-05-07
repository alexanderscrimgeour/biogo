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
	position    point
	blobSize    int
	foodBlobs   []*Blob
	blobs       []*Blob
	walls       []*Line
}

func NewRenderGrid(xPos, yPos float64, blobSize int) *RenderGrid {
	return &RenderGrid{position: point{X: xPos, Y: yPos}, blobSize: blobSize}
}

func (g *RenderGrid) DrawGrid(image *ebiten.Image) {
	for _, wall := range g.walls {
		wall.Draw(image)
	}
	for _, blob := range g.foodBlobs {
		blob.Draw(image)
	}
	for _, blob := range g.blobs {
		blob.Draw(image)
	}
}

// DrawBackground draws only the static elements (walls and food).
// Creatures are excluded so the caller can render them with interpolation.
func (g *RenderGrid) DrawBackground(image *ebiten.Image) {
	for _, wall := range g.walls {
		wall.Draw(image)
	}
	for _, blob := range g.foodBlobs {
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

func (g *RenderGrid) RemoveBlob(blob *Blob) {
	for i, b := range g.blobs {
		if b == blob {
			g.blobs = append(g.blobs[:i], g.blobs[i+1:]...)
			return
		}
	}
}

func (g *RenderGrid) AddFoodBlob(blobWidth int, c color.Color) *Blob {
	newImage := ebiten.NewImage(blobWidth, blobWidth)
	newImage.Fill(c)
	blob := NewBlob(newImage, &ebiten.GeoM{})
	g.foodBlobs = append(g.foodBlobs, blob)
	return blob
}

func (g *RenderGrid) RemoveFoodBlob(blob *Blob) {
	for i, b := range g.foodBlobs {
		if b == blob {
			g.foodBlobs = append(g.foodBlobs[:i], g.foodBlobs[i+1:]...)
			return
		}
	}
}
