package simulation

// CreatureView is a read-only snapshot of a creature's display state.
// It contains only what the rendering layer needs, keeping simulation internals private.
type CreatureView struct {
	ID         int
	X, Y       int
	R, G, B, A uint8
}

// CreatureViews returns a snapshot of all living creatures for rendering.
func (s *Simulation) CreatureViews() []CreatureView {
	views := make([]CreatureView, 0, len(s.Population.Creatures))
	for _, c := range s.Population.Creatures {
		r, g, b, a := genomeColor(c.Genome)
		views = append(views, CreatureView{
			ID: c.Id,
			X:  c.Loc.X,
			Y:  c.Loc.Y,
			R:  r, G: g, B: b, A: a,
		})
	}
	return views
}
