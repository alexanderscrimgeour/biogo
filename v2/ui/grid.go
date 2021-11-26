package ui

import (
	"image/color"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/colornames"
)

type point struct {
	X float64
	Y float64
}

type Grid struct {
	position point
	columns  int
	rows     int
	blobSize int

	blobs []*Blob

	gridLines []*Line
}

func NewGrid(xPos, yPos float64, rows, columns, blobSize int) *Grid {
	grid := &Grid{position: point{X: xPos, Y: yPos}, rows: rows, columns: columns, blobSize: blobSize}
	grid.generateGridLines()
	return grid
}

func (g *Grid) DrawGrid(image *ebiten.Image) {
	// for _, gridLine := range g.gridLines {
	// 	gridLine.Draw(image)
	// }
	for _, blob := range g.blobs {
		blob.Draw(image)
	}
}

func (g *Grid) generateGridLines() {
	lineWidth := BlockSize
	width := g.columns * g.blobSize
	height := g.rows * g.blobSize

	for x := g.position.X; x <= g.position.X+float64(width); x += float64(g.blobSize) {
		g.gridLines = append(g.gridLines, NewLine(lineWidth, height, x, g.position.Y))
	}

	for y := g.position.Y; y <= g.position.Y+float64(height); y += float64(g.blobSize) {
		g.gridLines = append(g.gridLines, NewLine(width, lineWidth, g.position.X, float64(y)))
	}
}

func (g *Grid) AddBlob(blobWidth int) *Blob {
	var newImage *ebiten.Image
	newImage = ebiten.NewImage(blobWidth, blobWidth)
	colors := []color.RGBA{colornames.Aliceblue,
		colornames.Antiquewhite,
		colornames.Aqua,
		colornames.Aquamarine,
		colornames.Azure,
		colornames.Beige,
		colornames.Bisque,
		colornames.Black,
		colornames.Blanchedalmond,
		colornames.Blue,
		colornames.Blueviolet,
		colornames.Brown,
		colornames.Burlywood,
		colornames.Cadetblue,
		colornames.Chartreuse,
		colornames.Chocolate,
		colornames.Coral,
		colornames.Cornflowerblue,
		colornames.Cornsilk,
		colornames.Crimson,
		colornames.Cyan,
		colornames.Darkblue,
		colornames.Darkcyan,
		colornames.Darkgoldenrod,
		colornames.Darkgray,
		colornames.Darkgreen,
		colornames.Darkgrey,
		colornames.Darkkhaki,
		colornames.Darkmagenta,
		colornames.Darkolivegreen,
		colornames.Darkorange,
		colornames.Darkorchid,
		colornames.Darkred,
		colornames.Darksalmon,
		colornames.Darkseagreen,
		colornames.Darkslateblue,
		colornames.Darkslategray,
		colornames.Darkslategrey,
		colornames.Darkturquoise,
		colornames.Darkviolet,
		colornames.Deeppink,
		colornames.Deepskyblue,
		colornames.Dimgray,
		colornames.Dimgrey,
		colornames.Dodgerblue,
		colornames.Firebrick,
		colornames.Floralwhite,
		colornames.Forestgreen,
		colornames.Fuchsia,
		colornames.Gainsboro,
		colornames.Ghostwhite,
		colornames.Gold,
		colornames.Goldenrod,
		colornames.Gray,
		colornames.Green,
		colornames.Greenyellow,
		colornames.Grey,
		colornames.Honeydew,
		colornames.Hotpink,
		colornames.Indianred,
		colornames.Indigo,
		colornames.Ivory,
		colornames.Khaki,
		colornames.Lavender,
		colornames.Lavenderblush,
		colornames.Lawngreen,
		colornames.Lemonchiffon,
		colornames.Lightblue,
		colornames.Lightcoral,
		colornames.Lightcyan,
		colornames.Lightgoldenrodyellow,
		colornames.Lightgray,
		colornames.Lightgreen,
		colornames.Lightgrey,
		colornames.Lightpink,
		colornames.Lightsalmon,
		colornames.Lightseagreen,
		colornames.Lightskyblue,
		colornames.Lightslategray,
		colornames.Lightslategrey,
		colornames.Lightsteelblue,
		colornames.Lightyellow,
		colornames.Lime,
		colornames.Limegreen,
		colornames.Linen,
		colornames.Magenta,
		colornames.Maroon,
		colornames.Mediumaquamarine,
		colornames.Mediumblue,
		colornames.Mediumorchid,
		colornames.Mediumpurple,
		colornames.Mediumseagreen,
		colornames.Mediumslateblue,
		colornames.Mediumspringgreen,
		colornames.Mediumturquoise,
		colornames.Mediumvioletred,
		colornames.Midnightblue,
		colornames.Mintcream,
		colornames.Mistyrose,
		colornames.Moccasin,
		colornames.Navajowhite,
		colornames.Navy,
		colornames.Oldlace,
		colornames.Olive,
		colornames.Olivedrab,
		colornames.Orange,
		colornames.Orangered,
		colornames.Orchid,
		colornames.Palegoldenrod,
		colornames.Palegreen,
		colornames.Paleturquoise,
		colornames.Palevioletred,
		colornames.Papayawhip,
		colornames.Peachpuff,
		colornames.Peru,
		colornames.Pink,
		colornames.Plum,
		colornames.Powderblue,
		colornames.Purple,
		colornames.Red,
		colornames.Rosybrown,
		colornames.Royalblue,
		colornames.Saddlebrown,
		colornames.Salmon,
		colornames.Sandybrown,
		colornames.Seagreen,
		colornames.Seashell,
		colornames.Sienna,
		colornames.Silver,
		colornames.Skyblue,
		colornames.Slateblue,
		colornames.Slategray,
		colornames.Slategrey,
		colornames.Snow,
		colornames.Springgreen,
		colornames.Steelblue,
		colornames.Tan,
		colornames.Teal,
		colornames.Thistle,
		colornames.Tomato,
		colornames.Turquoise,
		colornames.Violet,
		colornames.Wheat,
		colornames.White,
		colornames.Whitesmoke,
		colornames.Yellow,
		colornames.Yellowgreen}
	rand := rand.Intn(len(colors))
	newImage.Fill(colors[rand])
	blob := NewBlob(newImage, &ebiten.GeoM{})
	g.blobs = append(g.blobs, blob)
	return blob
}
