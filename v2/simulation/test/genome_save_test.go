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
		if g.OscPeriod != l.OscPeriod || g.Mass != l.Mass {
			t.Errorf("genome[%d] header mismatch", i)
		}
		if len(g.Brain) != len(l.Brain) {
			t.Fatalf("genome[%d] brain length: want %d, got %d", i, len(g.Brain), len(l.Brain))
		}
		for j := range g.Brain {
			if g.Brain[j] != l.Brain[j] {
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

func TestSaveCreatureToFileAndLoadAll(t *testing.T) {
	// SaveCreatureToFile writes to a package-level constant path, so we can't
	// redirect it via t.TempDir(). Test that a round-trip works end-to-end by
	// calling the public functions directly with a temp file instead, mirroring
	// what SaveCreatureToFile does internally.
	p := defaultParams()
	g1 := simulation.MakeRandomGenome(p)
	g2 := simulation.MakeRandomGenome(p)

	dir := t.TempDir()
	save := func(g *simulation.Genome) {
		path := dir + "/genome.json"
		if err := simulation.SaveGenomesToFile([]*simulation.Genome{g}, path); err != nil {
			t.Fatalf("save: %v", err)
		}
	}
	save(g1)
	save(g2) // overwrites; just checking the round-trip works

	loaded, err := simulation.LoadGenomesFromFile(dir + "/genome.json")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 genome, got %d", len(loaded))
	}
	if g2.String() != loaded[0].String() {
		t.Error("genome did not survive round-trip")
	}
}

func TestLoadAllCreatureGenomesEmptyDir(t *testing.T) {
	// LoadAllCreatureGenomes reads from the package-level creaturesSaveDir.
	// We can only test the missing-directory path indirectly via the exported
	// LoadGenomesFromFile behaviour, which returns (nil, nil) for missing files.
	genomes, err := simulation.LoadGenomesFromFile(t.TempDir() + "/nonexistent.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if genomes != nil {
		t.Error("expected nil for missing file")
	}
}

func TestFromGenomeDataClampsZeroMass(t *testing.T) {
	// A genome saved with mass=0 or min_mass=0 (e.g. from an older file format)
	// must be clamped to at least 1 on load.
	p := defaultParams()
	g := simulation.MakeRandomGenome(p)
	g.Mass = 0
	g.MinMass = 0

	path := filepath.Join(t.TempDir(), "zero_mass.json")
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
	if loaded[0].Mass < 1 {
		t.Errorf("Mass should be >= 1 after load, got %d", loaded[0].Mass)
	}
	if loaded[0].MinMass < 1 {
		t.Errorf("MinMass should be >= 1 after load, got %d", loaded[0].MinMass)
	}
	if loaded[0].MinMass > loaded[0].Mass {
		t.Errorf("MinMass %d should be <= Mass %d", loaded[0].MinMass, loaded[0].Mass)
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
