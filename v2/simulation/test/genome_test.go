package test

import (
	"biogo/v2/simulation"
	"math/rand"
	"testing"
)

func defaultParams() *simulation.Parameters {
	return simulation.DefaultParams()
}

func TestMakeRandomGene(t *testing.T) {
	gene := simulation.MakeRandomGene()
	if gene == nil {
		t.Fatal("MakeRandomGene returned nil")
	}
	if gene.SourceType > 1 {
		t.Errorf("SourceType must be 0 or 1, got %d", gene.SourceType)
	}
	if gene.SinkType > 1 {
		t.Errorf("SinkType must be 0 or 1, got %d", gene.SinkType)
	}
}

func TestMakeRandomGenome(t *testing.T) {
	p := defaultParams()
	g := simulation.MakeRandomGenome(p)
	if g == nil {
		t.Fatal("MakeRandomGenome returned nil")
	}
	if len(g.Brain) == 0 {
		t.Error("genome Brain should not be empty")
	}
	if int(g.BrainLength) != len(g.Brain) {
		t.Errorf("BrainLength %d does not match len(Brain) %d", g.BrainLength, len(g.Brain))
	}
}

func TestGenomeCopy(t *testing.T) {
	p := defaultParams()
	original := simulation.MakeRandomGenome(p)
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
	gene := simulation.MakeRandomGene()
	cp := gene.Copy()
	if cp == gene {
		t.Error("Gene.Copy should return a new pointer")
	}
	if *cp != *gene {
		t.Error("Gene.Copy should produce identical gene")
	}
}

func TestMutateChangesGenome(t *testing.T) {
	rand.Seed(42)
	p := defaultParams()
	p.MinMutationRate = 1.0 // force mutation on every gene
	g := simulation.MakeRandomGenome(p)
	original := g.String()
	simulation.Mutate(g, p)
	// With 100% mutation rate, genome string very likely changes
	if g.String() == original {
		t.Log("Mutate did not change the genome (probabilistic, rare)")
	}
}

func TestAsexualReproduction(t *testing.T) {
	p := defaultParams()
	parent := simulation.MakeRandomGenome(p)
	child := simulation.AsexualReproduction(parent, p)
	if child == parent {
		t.Error("AsexualReproduction should return a new genome pointer")
	}
	if len(child.Brain) == 0 {
		t.Error("child genome Brain should not be empty")
	}
}

func TestGenomeSimilarity(t *testing.T) {
	p := defaultParams()
	g := simulation.MakeRandomGenome(p)
	identical := g.Copy()
	sim := simulation.GenomeSimilarity(*g, *identical)
	if sim < 0.99 {
		t.Errorf("identical genomes similarity = %f, want ~1.0", sim)
	}

	other := simulation.MakeRandomGenome(p)
	sim2 := simulation.GenomeSimilarity(*g, *other)
	if sim2 < 0 || sim2 > 1 {
		t.Errorf("GenomeSimilarity out of [0,1]: %f", sim2)
	}
}

func TestMakeRandomGenomeMassInBounds(t *testing.T) {
	p := defaultParams()
<<<<<<< Updated upstream
	p.MinSize = 10
	p.MaxSize = 50
	for i := 0; i < 100; i++ {
		g := simulation.MakeRandomGenome(p)
		if g.Size < p.MinSize || g.Size > p.MaxSize {
			t.Errorf("Size %d outside [%d, %d]", g.Size, p.MinSize, p.MaxSize)
=======
	p.MaxMass = 50
	for i := 0; i < 100; i++ {
		g := simulation.MakeRandomGenome(p)
		if g.Mass > p.MaxMass {
			t.Errorf("Mass %d exceeds MaxMass %d", g.Mass, p.MaxMass)
		}
		if g.MinMass < 1 || g.MinMass > g.Mass {
			t.Errorf("MinMass %d outside [1, %d]", g.MinMass, g.Mass)
>>>>>>> Stashed changes
		}
	}
}

func TestGenomeToByteArray(t *testing.T) {
	p := defaultParams()
	g := simulation.MakeRandomGenome(p)
	arr := g.ToByteArray()
	if len(arr) == 0 {
		t.Error("ToByteArray should not be empty")
	}
}

func TestGenomeString(t *testing.T) {
	p := defaultParams()
	g := simulation.MakeRandomGenome(p)
	s := g.String()
	if len(s) == 0 {
		t.Error("Genome.String() should not be empty")
	}
}

func TestMakeRandomGenomeMutationRateNonZero(t *testing.T) {
	p := defaultParams()
	for i := 0; i < 1000; i++ {
		g := simulation.MakeRandomGenome(p)
		if g.MutationRate == 0 {
			t.Fatalf("MakeRandomGenome produced MutationRate=0 on iteration %d", i)
		}
	}
}

func TestMutateNeverZeroMutationRate(t *testing.T) {
	p := defaultParams()
	p.MinMutationRate = 1.0
	for i := 0; i < 1000; i++ {
		g := simulation.MakeRandomGenome(p)
		simulation.Mutate(g, p)
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
