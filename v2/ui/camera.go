package ui

import "github.com/hajimehoshi/ebiten/v2"

// Camera holds the world-space point shown at the viewport centre plus zoom.
type Camera struct {
	X, Y float64
	Zoom float64
}

// GeoM returns the transform to apply when drawing world-space content to screen.
// Sequence: translate origin to camera centre → scale by zoom → translate to screen centre.
func (c *Camera) GeoM(screenW, screenH float64) ebiten.GeoM {
	var m ebiten.GeoM
	m.Translate(-c.X, -c.Y)
	m.Scale(c.Zoom, c.Zoom)
	m.Translate(screenW/2, screenH/2)
	return m
}

// ScreenToWorld converts a screen-space position to world-space coordinates.
func (c *Camera) ScreenToWorld(sx, sy, screenW, screenH float64) (float64, float64) {
	return (sx-screenW/2)/c.Zoom + c.X, (sy-screenH/2)/c.Zoom + c.Y
}

// Clamp keeps the camera centre within [0, worldW] × [0, worldH].
func (c *Camera) Clamp(worldW, worldH float64) {
	if c.X < 0 {
		c.X = 0
	} else if c.X > worldW {
		c.X = worldW
	}
	if c.Y < 0 {
		c.Y = 0
	} else if c.Y > worldH {
		c.Y = worldH
	}
}
