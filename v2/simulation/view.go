package simulation

// CreatureView is a read-only snapshot of a creature's display state.
// It contains only what the rendering layer needs, keeping simulation internals private.
type CreatureView struct {
	ID            int
	X, Y          int
	R, G, B, A    uint8
	DirX, DirY    int
	SightDistance byte
	FieldOfView   byte
	Size          byte
}

// FoodView is a read-only snapshot of a food item's position for rendering.
type FoodView struct {
	X, Y int
}

// CorpseView is a read-only snapshot of a dead creature for rendering.
// EnergyFraction is the ratio of remaining energy to MaxEnergy (0..1), used for alpha.
type CorpseView struct {
	ID             int
	X, Y           int
	EnergyFraction float32
}

// CreatureViews returns a snapshot of all living creatures for rendering.
func (s *Simulation) CreatureViews() []CreatureView {
	views := make([]CreatureView, 0, len(s.Population.Creatures))
	for _, c := range s.Population.Creatures {
		if !c.Alive {
			continue
		}
		r, g, b, a := genomeColor(c.Genome)
		views = append(views, CreatureView{
			ID:            c.Id,
			X:             c.Loc.X,
			Y:             c.Loc.Y,
			R:             r, G: g, B: b, A: a,
			DirX:          c.LastMoveDir.X,
			DirY:          c.LastMoveDir.Y,
			SightDistance: c.Genome.SightDistance,
			FieldOfView:   c.Genome.FieldOfView,
			Size:          c.Genome.Size,
		})
	}
	return views
}

func (s *Simulation) CreatureMinSize() byte { return 1 }
func (s *Simulation) CreatureMaxSize() byte { return s.Params.MaxSize }

// FoodViews returns a snapshot of all current food locations for rendering.
func (s *Simulation) FoodViews() []FoodView {
	views := make([]FoodView, len(s.Grid.FoodLocations))
	for i, loc := range s.Grid.FoodLocations {
		views[i] = FoodView{X: loc.X, Y: loc.Y}
	}
	return views
}

// CorpseViews returns a snapshot of all decaying corpses for rendering.
func (s *Simulation) CorpseViews() []CorpseView {
	views := make([]CorpseView, 0)
	for _, c := range s.Population.Creatures {
		if c.Alive {
			continue
		}
		sizeE := float32(c.Genome.Size)
		frac := float32(0)
		if sizeE > 0 {
			frac = c.Energy / sizeE
		}
		if frac > 1 {
			frac = 1
		}
		views = append(views, CorpseView{
			ID:             c.Id,
			X:              c.Loc.X,
			Y:              c.Loc.Y,
			EnergyFraction: frac,
		})
	}
	return views
}
