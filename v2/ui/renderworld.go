package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type point struct {
	X float64
	Y float64
}

type RenderWorld struct {
	position point
	unitSize int
	walls    []*Line
}

func NewRenderWorld(xPos, yPos float64, unitSize int) *RenderWorld {
	return &RenderWorld{
		position: point{X: xPos, Y: yPos},
		unitSize: unitSize,
	}
}

func (g *RenderWorld) DrawBackground(image *ebiten.Image, camGeoM ebiten.GeoM) {
	for _, wall := range g.walls {
		wall.Draw(image, camGeoM)
	}
}

func (g *RenderWorld) AddLine(minX, minY, maxX, maxY float64) *Line {
	wall := NewLine(minX, minY, maxX, maxY)
	g.walls = append(g.walls, wall)
	return wall
}
