package simulation

import (
	"biogo/v2/simulation/snapshot"
	"biogo/v2/world"
	"bytes"
	"encoding/gob"
	"fmt"
	"image/color"
	"path/filepath"
)

const saveGameDir = "data/games"

// SavedGame is a reference to a game save file, returned by ListSavedGames.
type SavedGame struct {
	Name string
	Path string
}

// ListSavedGames returns metadata for all saves in data/games, newest-first.
func (s *Simulation) ListSavedGames() []SavedGame {
	infos, _ := snapshot.ListSaves(saveGameDir)
	out := make([]SavedGame, len(infos))
	for i, info := range infos {
		out[i] = SavedGame{Name: info.Name, Path: info.Path}
	}
	return out
}

// SaveGame captures the current state and writes it to data/games/{name}.biogosave.
// If a file with that name already exists a timestamp suffix is appended.
func (s *Simulation) SaveGame(name string) error {
	safe := sanitizeFilename(name)
	if safe == "" {
		return fmt.Errorf("save name cannot be empty")
	}
	filename := safe + ".biogosave"
	path := filepath.Join(saveGameDir, filename)
	dto := s.captureState()
	return snapshot.Save(dto, path)
}

// SaveGameTo captures the current state and writes it to the exact path provided.
// Used by the UI when overwriting an existing save.
func (s *Simulation) SaveGameTo(path string) error {
	dto := s.captureState()
	return snapshot.Save(dto, path)
}

// LoadGame replaces the running simulation with the state stored at path.
func (s *Simulation) LoadGame(path string) error {
	dto, err := snapshot.Load(path)
	if err != nil {
		return err
	}
	return s.restoreState(dto)
}

// captureState converts the live simulation into a snapshot.SimulationDTO.
func (s *Simulation) captureState() snapshot.SimulationDTO {
	// Encode Parameters via gob (all value types, no registration needed).
	var paramsBuf bytes.Buffer
	_ = gob.NewEncoder(&paramsBuf).Encode(*s.Params)

	// --- World fountains (flat parallel arrays, grouped foliage/fungi/meat) ---
	totalFountains := len(s.World.FoliageFountains) + len(s.World.FungiFountains) + len(s.World.MeatFountains)
	wdto := snapshot.WorldDTO{
		FountainCounts: [3]int{
			len(s.World.FoliageFountains),
			len(s.World.FungiFountains),
			len(s.World.MeatFountains),
		},
		FountainX: make([]float32, 0, totalFountains),
		FountainY: make([]float32, 0, totalFountains),
	}
	for _, f := range s.World.FoliageFountains {
		wdto.FountainX = append(wdto.FountainX, f.Pos.X)
		wdto.FountainY = append(wdto.FountainY, f.Pos.Y)
	}
	for _, f := range s.World.FungiFountains {
		wdto.FountainX = append(wdto.FountainX, f.Pos.X)
		wdto.FountainY = append(wdto.FountainY, f.Pos.Y)
	}
	for _, f := range s.World.MeatFountains {
		wdto.FountainX = append(wdto.FountainX, f.Pos.X)
		wdto.FountainY = append(wdto.FountainY, f.Pos.Y)
	}

	// --- Food (active items only, repacked) ---
	s.World.ForEachActiveFood(func(id int, x, y float64, _ float64, typ uint8) {
		wdto.FoodX = append(wdto.FoodX, float32(x))
		wdto.FoodY = append(wdto.FoodY, float32(y))
		wdto.FoodMass = append(wdto.FoodMass, s.World.GetFoodMass(id))
		wdto.FoodType = append(wdto.FoodType, typ)
	})

	// --- Population (alive only, repacked) ---
	aliveIDs := s.Population.AliveIDs()
	pdto := snapshot.PopulationDTO{
		Creatures: make([]snapshot.CreatureDTO, 0, len(aliveIDs)),
	}
	for _, id := range aliveIDs {
		c, ok := s.Population.Get(id)
		if !ok {
			continue
		}

		// Brain genes → parallel byte slices.
		brain := c.Genome.Brain
		bSrcID := make([]byte, len(brain))
		bSrcType := make([]byte, len(brain))
		bSinkID := make([]byte, len(brain))
		bSinkType := make([]byte, len(brain))
		bWeight := make([]byte, len(brain))
		for i, gn := range brain {
			bSrcID[i] = gn.SourceID
			bSrcType[i] = gn.SourceType
			bSinkID[i] = gn.SinkID
			bSinkType[i] = gn.SinkType
			bWeight[i] = gn.Weight
		}

		// Learned NNet weights and neuron states.
		weights := make([]float32, len(c.Nnet.Weights))
		copy(weights, c.Nnet.Weights)

		nOut := make([]float32, len(c.Nnet.HiddenNeurons))
		nAvg := make([]float32, len(c.Nnet.HiddenNeurons))
		nSens := make([]float32, len(c.Nnet.HiddenNeurons))
		for i, n := range c.Nnet.HiddenNeurons {
			nOut[i] = n.Output
			nAvg[i] = n.AverageOutput
			nSens[i] = n.Sensitivity
		}

		pdto.Creatures = append(pdto.Creatures, snapshot.CreatureDTO{
			Generation:                       c.Generation,
			Tier:                             c.Tier,
			Energy:                           c.Energy,
			Mass:                             c.Mass,
			SurvivalMass:                     c.SurvivalMass,
			Radius:                           c.Radius,
			VisionRadius:                     c.VisionRadius,
			Stomach:                          c.Stomach,
			Dopamine:                         c.Dopamine,
			LocX:                             c.Loc.X,
			LocY:                             c.Loc.Y,
			LastLocX:                         c.LastLoc.X,
			LastLocY:                         c.LastLoc.Y,
			BirthLocX:                        c.BirthLoc.X,
			BirthLocY:                        c.BirthLoc.Y,
			Heading:                          c.Heading,
			Speed:                            c.Speed,
			Responsiveness:                   c.Responsiveness,
			Clock:                            c.Clock,
			BaseOscTick:                      c.BaseOscTick,
			IsResting:                        c.IsResting,
			Age:                              c.Age,
			ReproductionCooldown:             c.ReproductionCooldown,
			ColorR:                           c.Color.R,
			ColorG:                           c.Color.G,
			ColorB:                           c.Color.B,
			ColorA:                           c.Color.A,
			GenomeOscPeriod:                  c.Genome.OscPeriod,
			GenomeVisionRadius:               c.Genome.VisionRadius,
			GenomeFieldOfView:                c.Genome.FieldOfView,
			GenomeResponsiveness:             c.Genome.Responsiveness,
			GenomeMutationRate:               c.Genome.MutationRate,
			GenomeBodyMass:                   c.Genome.BodyMass,
			GenomeSurvivalMass:               c.Genome.SurvivalMass,
			GenomeReproductionType:           c.Genome.ReproductionType,
			GenomeCognitiveBreadth:           c.Genome.CognitiveBreadth,
			GenomeSynapticDensity:            c.Genome.SynapticDensity,
			GenomeJuvenilePeriod:             c.Genome.JuvenilePeriod,
			GenomeMetabolicRate:              c.Genome.MetabolicRate,
			GenomeStomachSize:                c.Genome.StomachSize,
			GenomeNeuroplasticity:            c.Genome.Neuroplasticity,
			GenomeLearningThreshold:          c.Genome.LearningThreshold,
			GenomeMassSplitRatio:             c.Genome.MassSplitRatio,
			GenomeFoliageDigestionEfficiency: c.Genome.FoliageDigestionEfficiency,
			GenomeFungiDigestionEfficiency:   c.Genome.FungiDigestionEfficiency,
			GenomeMeatDigestionEfficiency:    c.Genome.MeatDigestionEfficiency,
			BrainSourceID:                    bSrcID,
			BrainSourceType:                  bSrcType,
			BrainSinkID:                      bSinkID,
			BrainSinkType:                    bSinkType,
			BrainWeight:                      bWeight,
			NNetWeights:                      weights,
			NNetNeuronOut:                    nOut,
			NNetNeuronAvg:                    nAvg,
			NNetNeuronSens:                   nSens,
		})
	}

	return snapshot.SimulationDTO{
		Version:    snapshot.FormatVersion,
		Tick:       s.Tick,
		Energy:     s.Energy,
		ParamsGob:  paramsBuf.Bytes(),
		World:      wdto,
		Population: pdto,
	}
}

// restoreState replaces the running simulation with the data from dto.
func (s *Simulation) restoreState(dto snapshot.SimulationDTO) error {
	// 1. Decode Parameters.
	var params Parameters
	if err := gob.NewDecoder(bytes.NewReader(dto.ParamsGob)).Decode(&params); err != nil {
		return fmt.Errorf("decode params: %w", err)
	}
	params.recomputeCachedFields()
	s.Params = &params
	InitResponseCurve(s.Params)

	// 2. Rebuild world from scratch with restored dimensions.
	s.World = world.NewWorld(params.World.Width, params.World.Height, 1)
	s.World.TempMin = params.Environment.TempMin
	s.World.TempMax = params.Environment.TempMax

	// 3. Restore fountain positions.
	wdto := dto.World
	nFol := wdto.FountainCounts[0]
	nFun := wdto.FountainCounts[1]
	nMeat := wdto.FountainCounts[2]
	// InitFountains creates slices with random positions; we overwrite immediately.
	s.World.InitFountains(nFol, nFun, nMeat,
		params.Food.Foliage.StationaryCount, params.Food.Fungi.StationaryCount, params.Food.Meat.StationaryCount)
	off := 0
	for i := 0; i < nFol && off < len(wdto.FountainX); i++ {
		s.World.FoliageFountains[i].Pos = world.Position{X: wdto.FountainX[off], Y: wdto.FountainY[off]}
		off++
	}
	for i := 0; i < nFun && off < len(wdto.FountainX); i++ {
		s.World.FungiFountains[i].Pos = world.Position{X: wdto.FountainX[off], Y: wdto.FountainY[off]}
		off++
	}
	for i := 0; i < nMeat && off < len(wdto.FountainX); i++ {
		s.World.MeatFountains[i].Pos = world.Position{X: wdto.FountainX[off], Y: wdto.FountainY[off]}
		off++
	}

	// 4. Restore food.
	for i, typ := range wdto.FoodType {
		pos := world.Position{X: wdto.FoodX[i], Y: wdto.FoodY[i]}
		mass := wdto.FoodMass[i]
		switch typ {
		case world.FoodTypeFoliage:
			s.World.AddFoliage(pos, mass)
		case world.FoodTypeFungi:
			s.World.AddFungi(pos, mass)
		case world.FoodTypeMeat:
			s.World.AddMeat(pos, mass)
		}
	}

	// 5. Restore population.
	pop := NewPopulation(s.Params)
	for _, cdto := range dto.Population.Creatures {
		brain := make([]Gene, len(cdto.BrainSourceID))
		for i := range brain {
			brain[i] = Gene{
				SourceID:   cdto.BrainSourceID[i],
				SourceType: cdto.BrainSourceType[i],
				SinkID:     cdto.BrainSinkID[i],
				SinkType:   cdto.BrainSinkType[i],
				Weight:     cdto.BrainWeight[i],
			}
		}
		g := &Genome{
			OscPeriod:                  cdto.GenomeOscPeriod,
			VisionRadius:               cdto.GenomeVisionRadius,
			FieldOfView:                cdto.GenomeFieldOfView,
			Responsiveness:             cdto.GenomeResponsiveness,
			MutationRate:               cdto.GenomeMutationRate,
			BodyMass:                   cdto.GenomeBodyMass,
			SurvivalMass:               cdto.GenomeSurvivalMass,
			ReproductionType:           cdto.GenomeReproductionType,
			CognitiveBreadth:           cdto.GenomeCognitiveBreadth,
			SynapticDensity:            cdto.GenomeSynapticDensity,
			JuvenilePeriod:             cdto.GenomeJuvenilePeriod,
			MetabolicRate:              cdto.GenomeMetabolicRate,
			StomachSize:                cdto.GenomeStomachSize,
			Neuroplasticity:            cdto.GenomeNeuroplasticity,
			LearningThreshold:          cdto.GenomeLearningThreshold,
			MassSplitRatio:             cdto.GenomeMassSplitRatio,
			FoliageDigestionEfficiency: cdto.GenomeFoliageDigestionEfficiency,
			FungiDigestionEfficiency:   cdto.GenomeFungiDigestionEfficiency,
			MeatDigestionEfficiency:    cdto.GenomeMeatDigestionEfficiency,
			Brain:                      brain,
		}
		g.recomputeBytes()

		// Reconstruct NNet topology from genome (deterministic for same genome).
		nnet := CreateNeuralNetworkFromGenome(g.Brain, g.CognitiveBreadth)

		// Apply learned weights if the edge count matches.
		if len(cdto.NNetWeights) == len(nnet.Weights) {
			copy(nnet.Weights, cdto.NNetWeights)
		}
		// Apply learned neuron states if the neuron count matches.
		if len(cdto.NNetNeuronOut) == len(nnet.HiddenNeurons) {
			for i := range nnet.HiddenNeurons {
				nnet.HiddenNeurons[i].Output = cdto.NNetNeuronOut[i]
				nnet.HiddenNeurons[i].AverageOutput = cdto.NNetNeuronAvg[i]
				nnet.HiddenNeurons[i].Sensitivity = cdto.NNetNeuronSens[i]
			}
		}

		pos := world.Position{X: cdto.LocX, Y: cdto.LocY}
		id := s.World.AddCreature(pos)

		c := &Creature{
			Id:                   id,
			Alive:                true,
			Generation:           cdto.Generation,
			Tier:                 cdto.Tier,
			Energy:               cdto.Energy,
			Mass:                 cdto.Mass,
			SurvivalMass:         cdto.SurvivalMass,
			Radius:               cdto.Radius,
			VisionRadius:         cdto.VisionRadius,
			Stomach:              cdto.Stomach,
			Dopamine:             cdto.Dopamine,
			Loc:                  pos,
			LastLoc:              world.Position{X: cdto.LastLocX, Y: cdto.LastLocY},
			BirthLoc:             world.Position{X: cdto.BirthLocX, Y: cdto.BirthLocY},
			Heading:              cdto.Heading,
			Speed:                cdto.Speed,
			Responsiveness:       cdto.Responsiveness,
			Clock:                cdto.Clock,
			BaseOscTick:          cdto.BaseOscTick,
			IsResting:            cdto.IsResting,
			Age:                  cdto.Age,
			ReproductionCooldown: cdto.ReproductionCooldown,
			Color:                color.RGBA{R: cdto.ColorR, G: cdto.ColorG, B: cdto.ColorB, A: cdto.ColorA},
			Genome:               g,
			Nnet:                 *nnet,
		}
		// Rebuild private cached constants from genome + params.
		c.initCachedFields(g, s.Params)

		pop.SetCreature(id, c)
		pop.AddAlive(id)
	}
	s.Population = pop

	s.Tick = dto.Tick
	s.Energy = dto.Energy
	s.cacheDirty = true
	return nil
}
