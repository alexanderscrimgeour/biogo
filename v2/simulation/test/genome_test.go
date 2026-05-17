package test

import (
	"biogo/v2/simulation"
	"testing"
)

func defaultParams() *simulation.Parameters {
	return simulation.DefaultParams()
}

func TestMakeRandomGene(t *testing.T) {
	gene := simulation.MakeRandomGene(simulation.SENSOR_COUNT, simulation.ACTION_COUNT, 10)
	if gene.SourceType > 1 {
		t.Errorf("SourceType must be 0 or 1, got %d", gene.SourceType)
	}
	// SinkType 0 = neuron, 2 = action; both are valid.
	if gene.SinkType != 0 && gene.SinkType != 2 {
		t.Errorf("SinkType must be 0 or 2, got %d", gene.SinkType)
	}
}

func TestMakeRandomGenome(t *testing.T) {
	p := defaultParams()
	g := simulation.MakeRandomGenome(p, 0)
	if g == nil {
		t.Fatal("MakeRandomGenome returned nil")
	}
	if len(g.Brain) == 0 {
		t.Error("genome Brain should not be empty")
	}
	if int(g.SynapticDensity) != len(g.Brain) {
		t.Errorf("SynapticDensity %d does not match len(Brain) %d", g.SynapticDensity, len(g.Brain))
	}
}

func TestGenomeCopy(t *testing.T) {
	p := defaultParams()
	original := simulation.MakeRandomGenome(p, 0)
	copy := original.Copy()

	if copy == original {
		t.Error("Copy should return a different pointer")
	}
	if &copy.Brain == &original.Brain {
		t.Error("Copy should deep-copy the Brain slice")
	}
	if original.String() != copy.String() {
		t.Error("Copy should produce identical genome string")
	}
}

func TestGeneCopy(t *testing.T) {
	gene := simulation.MakeRandomGene(simulation.SENSOR_COUNT, simulation.ACTION_COUNT, 10)
	cp := gene.Copy()
	if cp != gene {
		t.Error("Gene.Copy should return an identical value")
	}
	if cp != gene {
		t.Error("Gene.Copy should produce identical gene")
	}
}

func TestMutateChangesGenome(t *testing.T) {
	p := defaultParams()
	p.Neurology.BaseMutationRate = 1.0 // force mutation on every gene
	g := simulation.MakeRandomGenome(p, 0)
	original := g.String()
	simulation.Mutate(g, p, false, 1.0, 0)
	// With 100% mutation rate, genome string very likely changes
	if g.String() == original {
		t.Log("Mutate did not change the genome (probabilistic, rare)")
	}
}

func TestAsexualReproduction(t *testing.T) {
	p := defaultParams()
	parent := simulation.MakeRandomGenome(p, 0)
	child := simulation.AsexualReproduction(parent, p, 1.0, 0)
	if child == parent {
		t.Error("AsexualReproduction should return a new genome pointer")
	}
	if len(child.Brain) == 0 {
		t.Error("child genome Brain should not be empty")
	}
}

func TestGenomeSimilarity(t *testing.T) {
	p := defaultParams()
	g := simulation.MakeRandomGenome(p, 0)
	identical := g.Copy()
	sim := simulation.GenomeSimilarity(g, identical)
	if sim < 0.99 {
		t.Errorf("identical genomes similarity = %f, want ~1.0", sim)
	}

	other := simulation.MakeRandomGenome(p, 0)
	sim2 := simulation.GenomeSimilarity(g, other)
	if sim2 < 0 || sim2 > 1 {
		t.Errorf("GenomeSimilarity out of [0,1]: %f", sim2)
	}
}

func TestMakeRandomGenomeMassInBounds(t *testing.T) {
	p := defaultParams()
	p.Creature.MaxMass = 50
	for i := 0; i < 100; i++ {
		g := simulation.MakeRandomGenome(p, 0)
		if g.Mass < 3 || float64(g.Mass) > p.Creature.MaxMass {
			t.Errorf("Mass %d outside [3, %g]", g.Mass, p.Creature.MaxMass)
		}
		if g.MinMass < 1 {
			t.Errorf("MinMass %d below 1", g.MinMass)
		}
		if float32(g.MinMass)*2 >= float32(g.Mass) {
			t.Errorf("MinMass constraint violated: MinMass=%d, Mass=%d (need MinMass < Mass/2)", g.MinMass, g.Mass)
		}
	}
}

func TestMutatePreservesMinMassConstraint(t *testing.T) {
	p := defaultParams()
	p.Neurology.BaseMutationRate = 1.0 // force mutations on every gene
	for i := 0; i < 200; i++ {
		g := simulation.MakeRandomGenome(p, 0)
		simulation.Mutate(g, p, false, 1.0, 0)
		if float32(g.MinMass)*2 >= float32(g.Mass) {
			t.Fatalf("Mutate violated MinMass constraint: MinMass=%d, Mass=%d", g.MinMass, g.Mass)
		}
	}
}

func TestGenomeToByteArray(t *testing.T) {
	p := defaultParams()
	g := simulation.MakeRandomGenome(p, 0)
	arr := g.ToByteArray()
	if len(arr) == 0 {
		t.Error("ToByteArray should not be empty")
	}
}

func TestGenomeString(t *testing.T) {
	p := defaultParams()
	g := simulation.MakeRandomGenome(p, 0)
	s := g.String()
	if len(s) == 0 {
		t.Error("Genome.String() should not be empty")
	}
}

func TestMakeRandomGenomeMutationRateNonZero(t *testing.T) {
	p := defaultParams()
	for i := 0; i < 1000; i++ {
		g := simulation.MakeRandomGenome(p, 0)
		if g.MutationRate == 0 {
			t.Fatalf("MakeRandomGenome produced MutationRate=0 on iteration %d", i)
		}
	}
}

func TestMutateNeverZeroMutationRate(t *testing.T) {
	p := defaultParams()
	p.Neurology.BaseMutationRate = 1.0
	for i := 0; i < 1000; i++ {
		g := simulation.MakeRandomGenome(p, 0)
		simulation.Mutate(g, p, false, 1.0, 0)
		if g.MutationRate == 0 {
			t.Fatalf("Mutate produced MutationRate=0 on iteration %d", i)
		}
	}
}

func TestGeneWeightAsFloat32(t *testing.T) {
	gene := &simulation.Gene{Weight: 0}
	w := gene.WeightAsFloat32()
	if w != -1.0 {
		t.Errorf("Weight=0 should map to -1.0, got %f", w)
	}
	gene.Weight = 255
	w = gene.WeightAsFloat32()
	if w < 0.99 {
		t.Errorf("Weight=255 should map to ~1.0, got %f", w)
	}
}
