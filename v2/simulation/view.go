package simulation

import "sort"

// CreatureView is a read-only snapshot of a creature's display state.
type CreatureView struct {
	ID            int
	X, Y          float64
	R, G, B, A    uint8
	Heading       float64 // radians
	SightDistance byte
	FieldOfView   byte
	Mass          byte
	CurrentMass   float64
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
	IsCorpse       bool
	EnergyFraction float32
}

// NNEdgeView is a single weighted connection in the neural network snapshot.
type NNEdgeView struct {
	SourceType byte
	SourceID   byte
	SinkType   byte
	SinkID     byte
	Weight     float32 // current learned weight
}

// NeuralNetView is a snapshot of a creature's neural network topology for rendering.
type NeuralNetView struct {
	Edges           []NNEdgeView
	HiddenNeuronIDs []byte // sorted active hidden neuron IDs
	SensorValues    map[byte]float32
	ActionValues    map[byte]float32
}

// CreatureDetailView is a rich snapshot of a single creature for the inspector panel.
type CreatureDetailView struct {
	ID              int
	Energy          float32
	MaxEnergy       float32
	Age             int
	IsJuvenile      bool
	JuvenilePeriod  int
	CurrentMass     float32
	AdultMass       byte
	LastAction      string
	SightDistance   byte
	FieldOfView     byte
	NeuronCount     byte
	BrainLength     int
	Dopamine        float32
	MutationPct     float32 // actual per-gene mutation probability as a percentage
	R, G, B, A      uint8   // genome-derived display colour
	MetabolicRate   float32 // energy drained per tick
	MaxAge          int     // maximum lifespan in ticks
	Stomach         float32
	StomachCapacity float32
	NeuralNet       NeuralNetView
}

// CreatureDetail returns a detailed view of a living creature by ID.
// The second return value is false if the creature is dead or not found.
func (s *Simulation) CreatureDetail(id int) (CreatureDetailView, bool) {
	c, ok := s.Population.Creatures[id]
	if !ok || !c.Alive {
		return CreatureDetailView{}, false
	}
	r, g, b, a := c.Color.RGBA()

	nnView := NeuralNetView{}
	for i, edge := range c.Nnet.Edges {
		w := edge.WeightAsFloat32()
		if i < len(c.Nnet.Weights) {
			w = c.Nnet.Weights[i]
		}
		nnView.Edges = append(nnView.Edges, NNEdgeView{
			SourceType: edge.SourceType,
			SourceID:   edge.SourceID,
			SinkType:   edge.SinkType,
			SinkID:     edge.SinkID,
			Weight:     w,
		})
	}
	for id := range c.Nnet.HiddenNeurons {
		nnView.HiddenNeuronIDs = append(nnView.HiddenNeuronIDs, id)
	}
	sort.Slice(nnView.HiddenNeuronIDs, func(i, j int) bool {
		return nnView.HiddenNeuronIDs[i] < nnView.HiddenNeuronIDs[j]
	})
	if c.Nnet.LastSensorValues != nil {
		nnView.SensorValues = make(map[byte]float32, len(c.Nnet.LastSensorValues))
		for k, v := range c.Nnet.LastSensorValues {
			nnView.SensorValues[k] = v
		}
	}
	if c.Nnet.LastActionValues != nil {
		nnView.ActionValues = make(map[byte]float32, len(c.Nnet.LastActionValues))
		for i, v := range c.Nnet.LastActionValues {
			nnView.ActionValues[byte(i)] = v
		}
	}

	return CreatureDetailView{
		ID:              c.Id,
		Energy:          c.Energy,
		MaxEnergy:       c.MaxEnergy(s.Params),
		Age:             c.Age,
		IsJuvenile:      c.IsJuvenile(s.Params),
		JuvenilePeriod:  c.JuvenilePeriod(s.Params),
		CurrentMass:     c.CurrentMass(s.Params),
		AdultMass:       c.Genome.Mass,
		LastAction:      c.LastAction,
		SightDistance:   c.Genome.SightDistance,
		FieldOfView:     c.Genome.FieldOfView,
		NeuronCount:     c.Genome.NeuronCount,
		BrainLength:     len(c.Genome.Brain),
		Dopamine:        c.Dopamine,
		MutationPct:     s.Params.MinMutationRate * float32(c.Genome.MutationRate) * 100,
		R:               uint8(r >> 8),
		G:               uint8(g >> 8),
		B:               uint8(b >> 8),
		A:               uint8(a >> 8),
		MetabolicRate:   c.MetabolicRate(s.Params),
		MaxAge:          c.MaxAge(s.Params),
		Stomach:         c.Stomach,
		StomachCapacity: c.StomachCapacity(s.Params),
		NeuralNet:       nnView,
	}, true
}

// CreatureViews returns a snapshot of all living creatures for rendering.
func (s *Simulation) CreatureViews() map[int]CreatureView {
	views := make(map[int]CreatureView, len(s.Population.Creatures))
	for _, c := range s.Population.Creatures {
		if !c.Alive {
			continue
		}
		r, g, b, a := c.Color.RGBA()
		views[c.Id] = CreatureView{
			ID: c.Id,
			X:  c.Loc.X,
			Y:  c.Loc.Y,
			R:  uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8),
			Heading:       c.Heading,
			SightDistance: c.Genome.SightDistance,
			FieldOfView:   c.Genome.FieldOfView,
			Mass:          c.Genome.Mass,
			CurrentMass:   float64(c.Mass),
		}
	}
	return views
}

func (s *Simulation) CreatureMinMass() byte { return 1 }
func (s *Simulation) CreatureMaxMass() byte { return s.Params.MaxMass }

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
		// Decay fraction: remaining mass relative to genome adult mass.
		frac := float32(0)
		if c.Genome.Mass > 0 {
			frac = c.Mass / float32(c.Genome.Mass)
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
