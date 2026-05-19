package ui

type CreatureAnim struct {
	PrevX, PrevY float64
	CurX, CurY   float64
	R, G, B, A   uint8
	Heading      float64
	Mass         float64
	Radius       float32
	Sexual       bool
	Tier         byte
}
