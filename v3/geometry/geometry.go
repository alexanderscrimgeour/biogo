package geometry

type Vector2 interface {
	Normal() float64
	Distance() float64
	Angle() float64
}

type Vector2Float64 struct {
	x float64
	y float64
}

// func (Vector2Float64 a) Normal(Vector2Float64 b) {
// 	result := float64(0.4)
	
// 	return result
// }

// func (Vector2Float64 a) Distance(Vector2Float64 b) {
// 	result := float64(0.4)
// 	// Calculate distance from a to b
// 	return result
// }

// func (Vector2Float64 a) Angle(Vector2Float64 b) {
// 	result := float64(0.4)
	
// 	return result
// }

