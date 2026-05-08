package components

import "image/color"

func lighten(c color.RGBA, factor float32) color.RGBA {
	return color.RGBA{
		R: uint8(float32(c.R) + (255-float32(c.R))*factor),
		G: uint8(float32(c.G) + (255-float32(c.G))*factor),
		B: uint8(float32(c.B) + (255-float32(c.B))*factor),
		A: c.A,
	}
}
