package simulation

import "biogo/v2/world"

// FoodTypeFoliage, FoodTypeFungi, and FoodTypeMeat mirror world.FoodType* for use in snapshot views.
const (
	FoodTypeFoliage = world.FoodTypeFoliage
	FoodTypeFungi   = world.FoodTypeFungi
	FoodTypeMeat    = world.FoodTypeMeat
)

// CreatureView is a read-only snapshot of a creature's display state.
type CreatureView struct {
	ID               int
	X, Y             float64
	R, G, B, A       uint8
	Heading          float64 // radians
	VisionRadius     float64
	FieldOfView      float64
	Radius           float64
	BodyMass         byte
	CurrentMass      float64
	ReproductionType byte // 0 = asexual, 1 = sexual
	Tier             byte
}

// FoodView is a read-only snapshot of a food item (foliage or meat) for rendering.
// Type is FoodTypeFoliage (0), FoodTypeMeat (1), or FoodTypeFungi (2).
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

// GenomeSnapshot is a copy of the raw genome header bytes for the inspector panel.
type GenomeSnapshot struct {
	OscPeriod                  byte
	VisionRadius               byte
	FieldOfView                byte
	Responsiveness             byte
	MutationRate               byte
	BodyMass                   byte
	SurvivalMass               byte
	ReproductionType           byte
	CognitiveBreadth           byte
	SynapticDensity            byte
	JuvenilePeriod             byte
	MetabolicRate              byte
	StomachSize                byte
	Neuroplasticity            byte
	LearningThreshold          byte
	MassSplitRatio             byte
	FoliageDigestionEfficiency byte
	FungiDigestionEfficiency   byte
	MeatDigestionEfficiency    byte
}

// CreatureDetailView is a rich snapshot of a single creature for the inspector panel.
type CreatureDetailView struct {
	ID                int
	Generation        float32
	Tier              byte
	Energy            float32
	MaxEnergy         float32
	Age               int
	IsJuvenile        bool
	JuvenilePeriod    int
	CurrentMass       float32
	LastAction        string
	VisionRadius      float64
	FieldOfView       float64
	Dopamine          float32
	MutationPct       float32 // actual per-gene mutation probability as a percentage
	R, G, B, A        uint8   // genome-derived display colour
	MetabolicRate     float32 // energy drained per tick
	MaxAge            int     // maximum lifespan in ticks
	Stomach           float64
	StomachCapacity   float64
	FoliageEfficiency float32 // normalised foliage digestion efficiency [0, 1]
	FungiEfficiency   float32 // normalised fungi digestion efficiency [0, 1]
	MeatEfficiency    float32 // normalised meat digestion efficiency [0, 1]
	ReproductionType  byte    // 0 = asexual, 1 = sexual
	Responsiveness    float32 // current responsiveness multiplier [-1, 1]
	NeuralNet         NeuralNetView
	Genome            GenomeSnapshot
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

	return CreatureDetailView{
		ID:                c.Id,
		Generation:        c.Generation,
		Tier:              c.Tier,
		Energy:            c.Energy,
		MaxEnergy:         c.MaxEnergy(s.Params),
		Age:               c.Age,
		IsJuvenile:        c.IsJuvenile(),
		JuvenilePeriod:    c.JuvenilePeriod(),
		CurrentMass:       float32(c.CurrentMass()),
		LastAction:        actionMaskToString(c.LastActionMask),
		VisionRadius:      float64(c.GetVisionRadius()),
		FieldOfView:       c.FieldOfView(),
		Dopamine:          c.Dopamine,
		MutationPct:       s.Params.Neurology.BaseMutationRate * float32(c.Genome.MutationRate),
		R:                 uint8(r >> 8),
		G:                 uint8(g >> 8),
		B:                 uint8(b >> 8),
		A:                 uint8(a >> 8),
		MetabolicRate:     c.MetabolicRate(s.Params, s.World.TemperatureAt(c.Loc.Y)),
		MaxAge:            c.MaxAge(s.Params),
		Stomach:           float64(c.Stomach),
		StomachCapacity:   float64(c.StomachCapacity(s.Params)),
		FoliageEfficiency: c.GetFoodEfficiency(FoodTypeFoliage),
		FungiEfficiency:   c.GetFoodEfficiency(FoodTypeFungi),
		MeatEfficiency:    c.GetFoodEfficiency(FoodTypeMeat),
		ReproductionType:  c.Genome.ReproductionType,
		Responsiveness:    c.Responsiveness,
		NeuralNet:         nnView,
		Genome: GenomeSnapshot{
			OscPeriod:                  c.Genome.OscPeriod,
			VisionRadius:               c.Genome.VisionRadius,
			FieldOfView:                c.Genome.FieldOfView,
			Responsiveness:             c.Genome.Responsiveness,
			MutationRate:               c.Genome.MutationRate,
			BodyMass:                   c.Genome.BodyMass,
			SurvivalMass:               c.Genome.SurvivalMass,
			ReproductionType:           c.Genome.ReproductionType,
			CognitiveBreadth:           c.Genome.CognitiveBreadth,
			SynapticDensity:            c.Genome.SynapticDensity,
			JuvenilePeriod:             c.Genome.JuvenilePeriod,
			MetabolicRate:              c.Genome.MetabolicRate,
			StomachSize:                c.Genome.StomachSize,
			Neuroplasticity:            c.Genome.Neuroplasticity,
			LearningThreshold:          c.Genome.LearningThreshold,
			MassSplitRatio:             c.Genome.MassSplitRatio,
			FoliageDigestionEfficiency: c.Genome.FoliageDigestionEfficiency,
			FungiDigestionEfficiency:   c.Genome.FungiDigestionEfficiency,
			MeatDigestionEfficiency:    c.Genome.MeatDigestionEfficiency,
		},
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
			VisionRadius:     float64(c.GetVisionRadius()),
			FieldOfView:      c.FieldOfView(),
			Radius:           float64(c.Radius),
			BodyMass:         c.Genome.BodyMass,
			CurrentMass:      float64(c.Mass),
			ReproductionType: c.Genome.ReproductionType,
			Tier:             c.Tier,
		})
	}
	return views
}
