package simulation

import "biogo/v2/world"

// FoodTypePlant and FoodTypeMeat mirror world.FoodType* for use in snapshot views.
const (
	FoodTypePlant = world.FoodTypePlant
	FoodTypeMeat  = world.FoodTypeMeat
)

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
	Tier             byte
}

// FoodView is a read-only snapshot of a food item (plant or meat) for rendering.
// Type is FoodTypePlant (0) or FoodTypeMeat (1).
type FoodView struct {
	ID     int
	X, Y   float64
	Radius float64
	Type   uint8
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
	Generation       float32
	Tier             byte
	Energy           float32
	MaxEnergy        float32
	Age              int
	IsJuvenile       bool
	JuvenilePeriod   int
	CurrentMass      float32
	AdultMass        float64
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
	FoodEfficiency   float32 // fraction of food mass absorbed per bite [0, 1]
	MeatEfficiency   float32 // fraction of meat mass absorbed per bite [0, 1]
	ReproductionType byte    // 0 = asexual, 1 = sexual
	Responsiveness   float32 // current responsiveness multiplier [-1, 1]
	NeuralNet        NeuralNetView
}

// CreatureDetail returns a detailed view of a living creature by ID.
// The second return value is false if the creature is dead or not found.
func (s *Simulation) CreatureDetail(id int) (CreatureDetailView, bool) {
	c, ok := s.Population.Get(id)
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
	for i := range c.Nnet.HiddenNeurons {
		nnView.HiddenNeuronIDs = append(nnView.HiddenNeuronIDs, byte(i))
	}
	nnView.SensorValues = make(map[byte]float32, SENSOR_COUNT)
	for sid := byte(0); sid < SENSOR_COUNT; sid++ {
		if c.Nnet.ActiveSensors[sid] {
			nnView.SensorValues[sid] = c.Nnet.LastSensorValues[sid]
		}
	}
	nnView.ActionValues = make(map[byte]float32, ACTION_COUNT)
	for i, v := range c.Nnet.LastActionValues {
		nnView.ActionValues[byte(i)] = v
	}

	foodEff, meatEff := c.DigestionEfficiencies()

	return CreatureDetailView{
		ID:               c.Id,
		Generation:       c.Generation,
		Tier:             c.Tier,
		Energy:           c.Energy,
		MaxEnergy:        c.MaxEnergy(s.Params),
		Age:              c.Age,
		IsJuvenile:       c.IsJuvenile(),
		JuvenilePeriod:   c.JuvenilePeriod(),
		CurrentMass:      float32(c.CurrentMass()),
		AdultMass:        float64(c.MaxMass),
		LastAction:       c.LastAction,
		SightDistance:    float64(c.GetSightDistance()),
		FieldOfView:      c.FieldOfView(),
		Dopamine:         c.Dopamine,
		MutationPct:      s.Params.BaseMutationRate * float32(c.Genome.MutationRate),
		R:                uint8(r >> 8),
		G:                uint8(g >> 8),
		B:                uint8(b >> 8),
		A:                uint8(a >> 8),
		MetabolicRate:    c.MetabolicRate(s.Params, s.World.TemperatureAt(c.Loc.Y)),
		MaxAge:           c.MaxAge(s.Params),
		Stomach:          float64(c.Stomach),
		StomachCapacity:  float64(c.StomachCapacity(s.Params)),
		FoodEfficiency:   foodEff,
		MeatEfficiency:   meatEff,
		ReproductionType: c.Genome.ReproductionType,
		Responsiveness:   c.Responsiveness,
		NeuralNet:        nnView,
	}, true
}

// CreatureViews returns a snapshot of all living creatures for rendering.
func (s *Simulation) CreatureViews() []CreatureView {
	views := make([]CreatureView, 0, len(s.Population.aliveIDs))
	for _, id := range s.Population.aliveIDs {
		c, ok := s.Population.Get(id)
		if !ok {
			continue
		}
		r, g, b, a := c.Color.RGBA()

		views = append(views, CreatureView{
			ID:               int(id),
			X:                float64(c.Loc.X),
			Y:                float64(c.Loc.Y),
			R:                uint8(r),
			G:                uint8(g),
			B:                uint8(b),
			A:                uint8(a),
			Heading:          float64(c.Heading),
			SightDistance:    float64(c.GetSightDistance()),
			FieldOfView:      c.FieldOfView(),
			Radius:           float64(c.Radius),
			Mass:             c.Genome.Mass,
			CurrentMass:      float64(c.Mass),
			ReproductionType: c.Genome.ReproductionType,
			Tier:             c.Tier,
		})
	}
	return views
}

func (s *Simulation) CreatureMinMass() byte    { return 1 }
func (s *Simulation) CreatureMaxMass() float64 { return s.Params.MaxMass }
