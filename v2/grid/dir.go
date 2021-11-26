package grid

import (
	"gopop/v2/utils"
	"math"
	"math/rand"
)

type Dir struct {
	X, Y int
}

var N = Dir{0, 1}
var NE = Dir{1, 1}
var E = Dir{1, 0}
var SE = Dir{1, -1}
var S = Dir{0, -1}
var SW = Dir{-1, -1}
var W = Dir{-1, 0}
var NW = Dir{-1, 1}
var CENTER = Dir{0, 0}

func (d Dir) Rotate90CW() Dir {
	return Dir{d.Y, -d.X}
}

func (d Dir) Rotate90CCW() Dir {
	return Dir{-d.Y, d.X}
}

func RandomDir() Dir {
	x := rand.Intn(3) - 1
	y := rand.Intn(3) - 1
	return Dir{X: x, Y: y}
}

func GetDirection(fromLoc, toLoc Coord) Dir {
	xDir := toLoc.X - fromLoc.X
	yDir := toLoc.Y - fromLoc.Y
	if xDir == 0 && yDir == 0 {
		return Dir{0, 0}
	}
	// TODO Make this better, this is inefficient
	if xDir > 0 {
		xDir = 1
	} else {
		xDir = -1
	}
	if yDir > 0 {
		yDir = 1
	} else {
		yDir = -1
	}

	return Dir{X: xDir, Y: yDir}
}

func RaySameness(fromDir, toDir Dir) float32 {
	fromMag := math.Sqrt(float64(fromDir.X*fromDir.X + fromDir.Y*fromDir.Y))
	toMag := math.Sqrt(float64(toDir.X*toDir.X + toDir.Y*toDir.Y))
	if fromMag == 0 || toMag == 0 {
		return 1
	}
	dot := float64(fromDir.X*toDir.X + fromDir.Y*toDir.Y)
	cos := float32(dot / (fromMag * toMag))
	cos = utils.MinFloat32(utils.MaxFloat32(cos, -1), 1)
	return cos
}
