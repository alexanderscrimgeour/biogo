package world

// SpatialHash partitions 2-D space into a flat array of cells, each holding a
// slice of integer IDs. Bucket lookup is a single multiply-add rather than a
// hash-map lookup, eliminating Go map hashing overhead for all spatial queries.
type SpatialHash struct {
	cells   [][]int
	numX    int
	numY    int
	invCell float64
}

// newSpatialHash builds a grid covering [0, width] × [0, height] with square
// cells of side length cellSize. Two extra cells on each axis prevent
// off-by-one bounds panics for positions right at the world edge.
func newSpatialHash(width, height, cellSize float64) *SpatialHash {
	numX := int(width/cellSize) + 2
	numY := int(height/cellSize) + 2
	return &SpatialHash{
		cells:   make([][]int, numX*numY),
		numX:    numX,
		numY:    numY,
		invCell: 1.0 / cellSize,
	}
}

// cellOf returns the (bx, by) grid coordinates for a world-space position,
// clamped to the valid cell range.
func (h *SpatialHash) cellOf(pos Position) (int, int) {
	bx := int(pos.X * h.invCell)
	by := int(pos.Y * h.invCell)
	if bx < 0 {
		bx = 0
	} else if bx >= h.numX {
		bx = h.numX - 1
	}
	if by < 0 {
		by = 0
	} else if by >= h.numY {
		by = h.numY - 1
	}
	return bx, by
}

func (h *SpatialHash) idx(bx, by int) int {
	return bx*h.numY + by
}

// Add registers id at the given position.
func (h *SpatialHash) Add(id int, pos Position) {
	bx, by := h.cellOf(pos)
	i := h.idx(bx, by)
	h.cells[i] = append(h.cells[i], id)
}

// Remove deletes id from the cell that contains pos via swap-and-truncate.
func (h *SpatialHash) Remove(id int, pos Position) {
	bx, by := h.cellOf(pos)
	i := h.idx(bx, by)
	s := h.cells[i]
	for j, v := range s {
		if v == id {
			s[j] = s[len(s)-1]
			h.cells[i] = s[:len(s)-1]
			return
		}
	}
}

// Move updates id from oldPos to newPos. When both positions fall in the same
// cell the function is a no-op, which is common for slow-moving entities.
func (h *SpatialHash) Move(id int, oldPos, newPos Position) {
	oldBx, oldBy := h.cellOf(oldPos)
	newBx, newBy := h.cellOf(newPos)
	if oldBx == newBx && oldBy == newBy {
		return
	}
	oldI := h.idx(oldBx, oldBy)
	s := h.cells[oldI]
	for j, v := range s {
		if v == id {
			s[j] = s[len(s)-1]
			h.cells[oldI] = s[:len(s)-1]
			break
		}
	}
	newI := h.idx(newBx, newBy)
	h.cells[newI] = append(h.cells[newI], id)
}

// cellBounds returns the clamped grid-cell bounding box for a circle query of
// halfExtent (radius or maxDist) centred at center.
func (h *SpatialHash) cellBounds(center Position, halfExtent float64) (minBx, maxBx, minBy, maxBy int) {
	minBx = int((center.X - halfExtent) * h.invCell)
	maxBx = int((center.X + halfExtent) * h.invCell)
	minBy = int((center.Y - halfExtent) * h.invCell)
	maxBy = int((center.Y + halfExtent) * h.invCell)
	if minBx < 0 {
		minBx = 0
	}
	if maxBx >= h.numX {
		maxBx = h.numX - 1
	}
	if minBy < 0 {
		minBy = 0
	}
	if maxBy >= h.numY {
		maxBy = h.numY - 1
	}
	return
}

// InRadius fills buffer with the IDs of all active entries within radius of
// center. positions[id] gives the entity's location; active[id] gates inclusion.
func (h *SpatialHash) InRadius(center Position, radius float64, positions []Position, active []bool, buffer []int) []int {
	buffer = buffer[:0]
	rSq := radius * radius
	minBx, maxBx, minBy, maxBy := h.cellBounds(center, radius)
	for bx := minBx; bx <= maxBx; bx++ {
		base := bx * h.numY
		for by := minBy; by <= maxBy; by++ {
			for _, id := range h.cells[base+by] {
				if id >= len(active) || !active[id] {
					continue
				}
				pos := positions[id]
				dx := pos.X - center.X
				dy := pos.Y - center.Y
				if dx*dx+dy*dy <= rSq {
					buffer = append(buffer, id)
				}
			}
		}
	}
	return buffer
}

// InCone fills buffer with IDs within maxDist and inside the forward cone
// defined by heading ± arccos(halfFOVCos). Entries at the exact center are excluded.
func (h *SpatialHash) InCone(center Position, heading, halfFOVCos, maxDist float64, positions []Position, active []bool, buffer []int) []int {
	buffer = buffer[:0]
	fwdX, fwdY := HeadingToVec(heading)
	rSq := maxDist * maxDist
	minBx, maxBx, minBy, maxBy := h.cellBounds(center, maxDist)
	for bx := minBx; bx <= maxBx; bx++ {
		base := bx * h.numY
		for by := minBy; by <= maxBy; by++ {
			for _, id := range h.cells[base+by] {
				if id >= len(active) || !active[id] {
					continue
				}
				pos := positions[id]
				dx := pos.X - center.X
				dy := pos.Y - center.Y
				if dx*dx+dy*dy > rSq {
					continue
				}
				if dx == 0 && dy == 0 {
					continue
				}
				if CosSimilarity(fwdX, fwdY, dx, dy) >= halfFOVCos {
					buffer = append(buffer, id)
				}
			}
		}
	}
	return buffer
}
