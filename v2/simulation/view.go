package simulation

// CreatureView is a read-only snapshot of a creature's display state.
type CreatureView struct {
	ID            int
	X, Y          float64
	R, G, B, A    uint8
	Heading       float64 // radians
	SightDistance byte
	FieldOfView   byte
	Mass          byte
}

// FoodView is a read-only snapshot of a food item's position for rendering.
type FoodView struct {
	ID   int
	X, Y float64
}

// CorpseView is a read-only snapshot of a dead creature for rendering.
type CorpseView struct {
	ID             int
	X, Y           float64
	EnergyFraction float32
}

// CreatureDetailView is a rich snapshot of a single creature for the inspector panel.
type CreatureDetailView struct {
	ID             int
	Energy         float32
	MaxEnergy      byte
	Age            int
	IsJuvenile     bool
	JuvenilePeriod int
	CurrentMass    float32
	AdultMass      byte
	LastAction     string
	SightDistance  byte
	FieldOfView    byte
	NeuronCount    byte
	BrainLength    int
	MutationPct    float32 // actual per-gene mutation probability as a percentage
}

// CreatureDetail returns a detailed view of a living creature by ID.
// The second return value is false if the creature is dead or not found.
func (s *Simulation) CreatureDetail(id int) (CreatureDetailView, bool) {
	c, ok := s.Population.Creatures[id]
	if !ok || !c.Alive {
		return CreatureDetailView{}, false
	}
	return CreatureDetailView{
		ID:             c.Id,
		Energy:         c.Energy,
		MaxEnergy:      c.Genome.MaxEnergy,
		Age:            c.Age,
		IsJuvenile:     c.IsJuvenile(s.Params),
		JuvenilePeriod: c.JuvenilePeriod(s.Params),
		CurrentMass:    c.CurrentMass(s.Params),
		AdultMass:      c.Genome.Mass,
		LastAction:     c.LastAction,
		SightDistance:  c.Genome.SightDistance,
		FieldOfView:    c.Genome.FieldOfView,
		NeuronCount:    c.Genome.NeuronCount,
		BrainLength:    len(c.Genome.Brain),
		MutationPct:    s.Params.MinMutationRate * float32(c.Genome.MutationRate) * 100,
	}, true
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
			Heading:       c.Heading,
			SightDistance: c.Genome.SightDistance,
			FieldOfView:   c.Genome.FieldOfView,
			Mass:          c.Genome.Mass,
		})
	}
	return views
}

<<<<<<< Updated upstream
func (s *Simulation) CreatureMinSize() byte { return s.Params.MinSize }
func (s *Simulation) CreatureMaxSize() byte { return s.Params.MaxSize }
=======
func (s *Simulation) CreatureMinMass() byte { return 1 }
func (s *Simulation) CreatureMaxMass() byte { return s.Params.MaxMass }
>>>>>>> Stashed changes

// FoodViews returns a snapshot of all current food locations for rendering.
func (s *Simulation) FoodViews() []FoodView {
	food := s.World.FoodPositions()
	views := make([]FoodView, 0, len(food))
	for id, pos := range food {
		views = append(views, FoodView{ID: id, X: pos.X, Y: pos.Y})
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
		sizeE := float32(c.Genome.Mass)
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
