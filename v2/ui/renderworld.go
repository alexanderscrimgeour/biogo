package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type point struct {
	X float64
	Y float64
}

type RenderWorld struct {
	position        point
	unitSize        int
	baseFoodImage   *ebiten.Image
	baseCorpseImage *ebiten.Image
	foodBlobs       []*Blob
	blobs           []*Blob
	walls           []*Line
}

func NewRenderWorld(xPos, yPos float64, unitSize int) *RenderWorld {
	foodImg := ebiten.NewImage(unitSize+10, unitSize+10)
	corpseImg := ebiten.NewImage(unitSize+10, unitSize+10)

	foodImg.Fill(color.Transparent)
	corpseImg.Fill(color.Transparent)

	c := float32(unitSize+10) / 2
	r := (float32(unitSize) / 2) - 0.5

	vector.DrawFilledCircle(foodImg, c, c, r, color.RGBA{65, 140, 55, 250}, true)
	vector.DrawFilledCircle(corpseImg, c, c, r, color.RGBA{120, 60, 20, 250}, true)

	return &RenderWorld{
		position:        point{X: xPos, Y: yPos},
		unitSize:        unitSize,
		baseFoodImage:   foodImg,
		baseCorpseImage: corpseImg,
	}
}

// DrawBackground draws only the static elements (walls and food).
// Creatures are excluded so the caller can render them with interpolation.
func (g *RenderWorld) DrawBackground(image *ebiten.Image) {
	for _, wall := range g.walls {
		wall.Draw(image)
	}
	for _, blob := range g.foodBlobs {
		blob.Draw(image)
	}
}

func (g *RenderWorld) AddLine(minX, minY, maxX, maxY float64) *Line {
	wall := NewLine(minX, minY, maxX, maxY)
	g.walls = append(g.walls, wall)
	return wall
}

func (g *RenderWorld) AddBlob(scale float64) *Blob {
	geoM := &ebiten.GeoM{}
	w := float64(g.baseFoodImage.Bounds().Dx())
	h := float64(g.baseFoodImage.Bounds().Dy())
	geoM.Translate(-w/2, -h/2)
	geoM.Scale(scale, scale)
	blob := NewBlob(g.baseCorpseImage, geoM)
	g.blobs = append(g.blobs, blob)
	return blob
}

func (g *RenderWorld) RemoveBlob(blob *Blob) {
	for i, b := range g.blobs {
		if b == blob {
			g.blobs = append(g.blobs[:i], g.blobs[i+1:]...)
			return
		}
	}
}

func (g *RenderWorld) AddFoodBlob(scale float64) *Blob {
	geoM := &ebiten.GeoM{}

	w := float64(g.baseFoodImage.Bounds().Dx())
	h := float64(g.baseFoodImage.Bounds().Dy())
	geoM.Translate(-w/2, -h/2)
	geoM.Scale(scale, scale)

	blob := NewBlob(g.baseFoodImage, geoM)
	g.foodBlobs = append(g.foodBlobs, blob)
	return blob
}
func (g *RenderWorld) RemoveFoodBlob(blob *Blob) {
	for i, b := range g.foodBlobs {
		if b == blob {
			g.foodBlobs = append(g.foodBlobs[:i], g.foodBlobs[i+1:]...)
			return
		}
	}
}
