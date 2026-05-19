package ui

import (
	"math"
)

type GeometryBuffer struct {
	UnitCircle       []struct{ x, y float32 }
	CircleIsTemplate []uint16
	VertsPerCircle   int
	IndicesPerCircle int
}

// NewGeometryBuffer pre-calculates the static index arrays and shapes
// used to feed the GPU streaming buffers.
func NewGeometryBuffer(segments int) *GeometryBuffer {
	vertsPerCircle := 1 + segments
	indicesPerCircle := 3 * segments

	const maxBatchCircles = 35000
	circleIsTemplate := make([]uint16, maxBatchCircles*indicesPerCircle)

	idxIdx := 0
	for i := 0; i < maxBatchCircles; i++ {
		baseVert := uint16(i * vertsPerCircle)
		for j := 0; j < segments; j++ {
			circleIsTemplate[idxIdx] = baseVert
			circleIsTemplate[idxIdx+1] = baseVert + 1 + uint16(j)

			if j == segments-1 {
				circleIsTemplate[idxIdx+2] = baseVert + 1
			} else {
				circleIsTemplate[idxIdx+2] = baseVert + 2 + uint16(j)
			}
			idxIdx += 3
		}
	}

	unitCircle := make([]struct{ x, y float32 }, segments)
	for i := 0; i < segments; i++ {
		angle := float64(i) * 2.0 * math.Pi / float64(segments)
		unitCircle[i].x = float32(math.Cos(angle))
		unitCircle[i].y = float32(math.Sin(angle))
	}

	return &GeometryBuffer{
		VertsPerCircle:   vertsPerCircle,
		IndicesPerCircle: indicesPerCircle,
		CircleIsTemplate: circleIsTemplate,
		UnitCircle:       unitCircle,
	}
}
