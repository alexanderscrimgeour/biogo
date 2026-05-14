package simulation

// CreatureView is a read-only snapshot of a creature's display state.
type CreatureView struct {
	ID               int
	X, Y             float64
	R, G, B, A       uint8
	Heading          float64 // radians
	SightDistance    float64
	FieldOfView      float64
	Radius           float64
	Mass             byte
	CurrentMass      float64
	ReproductionType byte // 0 = asexual, 1 = sexual
}

// FoodView is a read-only snapshot of a food item's position and mass for rendering.
type FoodView struct {
	ID     int
	X, Y   float64
	Radius float64
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
	ID               int
	Energy           float32
	MaxEnergy        float32
	Age              int
	IsJuvenile       bool
	JuvenilePeriod   int
	CurrentMass      float32
	AdultMass        byte
	LastAction       string
	SightDistance    float64
	FieldOfView      float64
	Dopamine         float32
	MutationPct      float32 // actual per-gene mutation probability as a percentage
	R, G, B, A       uint8   // genome-derived display colour
	MetabolicRate    float32 // energy drained per tick
	MaxAge           int     // maximum lifespan in ticks
	Stomach          float64
	StomachCapacity  float64
	ReproductionType byte // 0 = asexual, 1 = sexual
	NeuralNet        NeuralNetView
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
	nnView.HiddenNeuronIDs = append(nnView.HiddenNeuronIDs, c.Nnet.HiddenNeuronIDs...)
	nnView.SensorValues = make(map[byte]float32, SENSOR_COUNT)
	for sid := byte(0); sid < SENSOR_COUNT; sid++ {
		if c.Nnet.ActiveSensors[sid] {
			nnView.SensorValues[sid] = c.Nnet.LastSensorValues[sid]
		}
	}
	if c.Nnet.LastActionValues != nil {
		nnView.ActionValues = make(map[byte]float32, len(c.Nnet.LastActionValues))
		for i, v := range c.Nnet.LastActionValues {
			nnView.ActionValues[byte(i)] = v
		}
	}

	return CreatureDetailView{
		ID:               c.Id,
		Energy:           c.Energy,
		MaxEnergy:        c.MaxEnergy(s.Params),
		Age:              c.Age,
		IsJuvenile:       c.IsJuvenile(s.Params),
		JuvenilePeriod:   c.JuvenilePeriod(s.Params),
		CurrentMass:      c.CurrentMass(),
		AdultMass:        c.Genome.Mass,
		LastAction:       c.LastAction,
		SightDistance:    c.GetSightDistance(),
		FieldOfView:      c.FieldOfView(),
		Dopamine:         c.Dopamine,
		MutationPct:      s.Params.BaseMutationRate * float32(c.Genome.MutationRate),
		R:                uint8(r >> 8),
		G:                uint8(g >> 8),
		B:                uint8(b >> 8),
		A:                uint8(a >> 8),
		MetabolicRate:    c.MetabolicRate(s.Params, s.World.TemperatureAt(c.Loc.Y)),
		MaxAge:           c.MaxAge(s.Params),
		Stomach:          c.Stomach,
		StomachCapacity:  c.StomachCapacity(s.Params),
		ReproductionType: c.Genome.ReproductionType,
		NeuralNet:        nnView,
	}, true
}

// CreatureViews returns a snapshot of all living creatures for rendering.
func (s *Simulation) CreatureViews() []CreatureView {
	views := make([]CreatureView, 0, len(s.Population.aliveIDs))
	for _, id := range s.Population.aliveIDs {
		c, ok := s.Population.Creatures[id]
		if !ok {
			continue
		}
		r, g, b, a := c.Color.RGBA()

		views = append(views, CreatureView{
			ID:               int(id),
			X:                c.Loc.X,
			Y:                c.Loc.Y,
			R:                uint8(r),
			G:                uint8(g),
			B:                uint8(b),
			A:                uint8(a),
			Heading:          c.Heading,
			SightDistance:    c.GetSightDistance(),
			FieldOfView:      c.FieldOfView(),
			Radius:           c.Radius,
			Mass:             c.Genome.Mass,
			CurrentMass:      float64(c.Mass),
			ReproductionType: c.Genome.ReproductionType,
		})
	}
	return views
}

func (s *Simulation) CreatureMinMass() byte { return 1 }
func (s *Simulation) CreatureMaxMass() byte { return s.Params.MaxMass }
