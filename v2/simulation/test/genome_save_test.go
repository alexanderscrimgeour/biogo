package test

import (
	"biogo/v2/simulation"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadGenomes(t *testing.T) {
	p := defaultParams()
	genomes := []*simulation.Genome{
		simulation.MakeRandomGenome(p),
		simulation.MakeRandomGenome(p),
		simulation.MakeRandomGenome(p),
	}

	path := filepath.Join(t.TempDir(), "genomes.json")
	if err := simulation.SaveGenomesToFile(genomes, path); err != nil {
		t.Fatalf("SaveGenomesToFile: %v", err)
	}

	loaded, err := simulation.LoadGenomesFromFile(path)
	if err != nil {
		t.Fatalf("LoadGenomesFromFile: %v", err)
	}
	if len(loaded) != len(genomes) {
		t.Fatalf("expected %d genomes, got %d", len(genomes), len(loaded))
	}
	for i := range genomes {
		g, l := genomes[i], loaded[i]
		if g.OscPeriod != l.OscPeriod || g.MaxEnergy != l.MaxEnergy || g.Size != l.Size {
			t.Errorf("genome[%d] header mismatch", i)
		}
		if len(g.Brain) != len(l.Brain) {
			t.Fatalf("genome[%d] brain length: want %d, got %d", i, len(g.Brain), len(l.Brain))
		}
		for j := range g.Brain {
			if *g.Brain[j] != *l.Brain[j] {
				t.Errorf("genome[%d] brain[%d] mismatch", i, j)
			}
		}
	}
}

func TestLoadGenomesFileNotExist(t *testing.T) {
	genomes, err := simulation.LoadGenomesFromFile(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err != nil {
		t.Fatalf("expected nil error for missing file, got: %v", err)
	}
	if genomes != nil {
		t.Error("expected nil for missing file")
	}
}

func TestSelectBestGenomesEmptyPopulation(t *testing.T) {
	result := simulation.SelectBestGenomes(map[int]*simulation.Creature{})
	if result != nil {
		t.Errorf("expected nil for empty population, got %v", result)
	}
}

func TestSelectBestGenomesReturnsAtMostMax(t *testing.T) {
	p := defaultParams()
	p.StartingPopulation = 50
	sim := simulation.New(p)
	for i := 0; i < 5; i++ {
		sim.Update()
	}

	result := simulation.SelectBestGenomes(sim.Population.Creatures)
	if len(result) > simulation.MaxSavedGenomes {
		t.Errorf("expected at most %d genomes, got %d", simulation.MaxSavedGenomes, len(result))
	}
	for i, g := range result {
		if g == nil {
			t.Errorf("result[%d] is nil", i)
		}
		if len(g.Brain) == 0 {
			t.Errorf("result[%d] has empty brain", i)
		}
	}
}

func TestSavedGenomesSurviveRoundTrip(t *testing.T) {
	p := defaultParams()
	g := simulation.MakeRandomGenome(p)
	path := filepath.Join(t.TempDir(), "rt.json")

	if err := simulation.SaveGenomesToFile([]*simulation.Genome{g}, path); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := simulation.LoadGenomesFromFile(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 genome, got %d", len(loaded))
	}
	// String() encodes all fields; identical strings mean identical genomes.
	if g.String() != loaded[0].String() {
		t.Error("genome did not survive round-trip serialisation")
	}
}
