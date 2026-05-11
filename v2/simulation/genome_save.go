package simulation

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// MaxSavedGenomes is the number of representative genomes persisted by SaveBestGenomes.
const MaxSavedGenomes = 5

const creaturesSaveDir = "data/creatures"

const clusterSimilarityThreshold = float32(0.9)
const clusterMaxCount = MaxSavedGenomes * 4

type geneData struct {
	SourceType byte `json:"source_type"`
	SourceID   byte `json:"source_id"`
	SinkType   byte `json:"sink_type"`
	SinkID     byte `json:"sink_id"`
	Weight     byte `json:"weight"`
}

type genomeData struct {
	OscPeriod        byte       `json:"osc_period"`
	SightDistance    byte       `json:"sight_distance"`
	FieldOfView      byte       `json:"field_of_view"`
	Responsiveness   byte       `json:"responsiveness"`
	MutationRate     byte       `json:"mutation_rate"`
	Mass             byte       `json:"mass"`
	MinMass          byte       `json:"min_mass"`
	ReproductionType byte       `json:"reproduction_type"`
	CognitiveBreadth      byte       `json:"neuron_count"`
	SynapticDensity      byte       `json:"brain_length"`
	JuvenilePeriod   byte       `json:"juvenile_period"`
	MetabolicRate    byte       `json:"metabolic_rate"`
	StomachSize      byte       `json:"stomach_size"`
	Neuroplasticity     byte       `json:"learning_rate"`
	LearningThreshold byte      `json:"learning_threshold"`
	Brain            []geneData `json:"brain"`
}

func toGenomeData(g *Genome) genomeData {
	genes := make([]geneData, len(g.Brain))
	for i, gene := range g.Brain {
		genes[i] = geneData{
			SourceType: gene.SourceType,
			SourceID:   gene.SourceID,
			SinkType:   gene.SinkType,
			SinkID:     gene.SinkID,
			Weight:     gene.Weight,
		}
	}
	return genomeData{
		OscPeriod:        g.OscPeriod,
		SightDistance:    g.SightDistance,
		FieldOfView:      g.FieldOfView,
		Responsiveness:   g.Responsiveness,
		MutationRate:     g.MutationRate,
		Mass:             g.Mass,
		MinMass:          g.MinMass,
		ReproductionType: g.ReproductionType,
		CognitiveBreadth:      g.CognitiveBreadth,
		SynapticDensity:      g.SynapticDensity,
		JuvenilePeriod:   g.JuvenilePeriod,
		MetabolicRate:     g.MetabolicRate,
		StomachSize:       g.StomachSize,
		Neuroplasticity:      g.Neuroplasticity,
		LearningThreshold: g.LearningThreshold,
		Brain:             genes,
	}
}

func fromGenomeData(gd genomeData) *Genome {
	genes := make([]*Gene, len(gd.Brain))
	for i, entry := range gd.Brain {
		genes[i] = &Gene{
			SourceType: entry.SourceType,
			SourceID:   entry.SourceID,
			SinkType:   entry.SinkType,
			SinkID:     entry.SinkID,
			Weight:     entry.Weight,
		}
	}
	mass := gd.Mass
	if mass < 1 {
		mass = 1
	}
	minMass := gd.MinMass
	if minMass < 1 {
		minMass = 1
	}
	if minMass > mass {
		minMass = mass
	}
	g := &Genome{
		OscPeriod:         gd.OscPeriod,
		SightDistance:     gd.SightDistance,
		FieldOfView:       gd.FieldOfView,
		Responsiveness:    gd.Responsiveness,
		MutationRate:      gd.MutationRate,
		Mass:              mass,
		MinMass:           minMass,
		ReproductionType:  gd.ReproductionType,
		CognitiveBreadth:  gd.CognitiveBreadth,
		SynapticDensity:   gd.SynapticDensity,
		JuvenilePeriod:    gd.JuvenilePeriod,
		MetabolicRate:     gd.MetabolicRate,
		StomachSize:       gd.StomachSize,
		Neuroplasticity:   gd.Neuroplasticity,
		LearningThreshold: gd.LearningThreshold,
		Brain:             genes,
	}
	g.recomputeBytes()
	return g
}

type genomeCluster struct {
	genome *Genome
	count  int
}

// SelectBestGenomes returns up to MaxSavedGenomes representative genomes from
// the current population via greedy similarity clustering. Clusters with the
// most members come first.
func SelectBestGenomes(creatures map[int]*Creature) []*Genome {
	alive := sampleAlive(creatures)
	if len(alive) == 0 {
		return nil
	}

	var clusters []genomeCluster
	for _, c := range alive {
		bestIdx := -1
		bestSim := float32(-1)
		for i, cl := range clusters {
			if sim := GenomeSimilarity(c.Genome, cl.genome); sim > bestSim {
				bestSim = sim
				bestIdx = i
			}
		}
		switch {
		case bestIdx >= 0 && bestSim >= clusterSimilarityThreshold:
			clusters[bestIdx].count++
		case len(clusters) < clusterMaxCount:
			clusters = append(clusters, genomeCluster{genome: c.Genome, count: 1})
		case bestIdx >= 0:
			clusters[bestIdx].count++
		}
	}

	sort.Slice(clusters, func(i, j int) bool { return clusters[i].count > clusters[j].count })

	n := MaxSavedGenomes
	if len(clusters) < n {
		n = len(clusters)
	}
	result := make([]*Genome, n)
	for i := range result {
		result[i] = clusters[i].genome
	}
	return result
}

// sampleAlive returns a random sample of alive creatures with non-empty brains,
// capped at 200 to bound the clustering cost.
func sampleAlive(creatures map[int]*Creature) []*Creature {
	const maxSample = 200
	alive := make([]*Creature, 0, len(creatures))
	for _, c := range creatures {
		if c.Alive && len(c.Genome.Brain) > 0 {
			alive = append(alive, c)
		}
	}
	if len(alive) <= maxSample {
		return alive
	}
	perm := rand.Perm(len(alive))
	sample := make([]*Creature, maxSample)
	for i := range sample {
		sample[i] = alive[perm[i]]
	}
	return sample
}

// SaveGenomesToFile serialises genomes to JSON at path, creating parent directories as needed.
func SaveGenomesToFile(genomes []*Genome, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data := make([]genomeData, len(genomes))
	for i, g := range genomes {
		data[i] = toGenomeData(g)
	}
	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, encoded, 0644)
}

// SaveCreatureToFile saves a single genome as a uniquely-named JSON file in data/creatures/.
func SaveCreatureToFile(g *Genome) error {
	if err := os.MkdirAll(creaturesSaveDir, 0755); err != nil {
		return err
	}
	name := fmt.Sprintf("%d_%d.json", time.Now().UnixNano(), rand.Int63())
	path := filepath.Join(creaturesSaveDir, name)
	data := toGenomeData(g)
	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, encoded, 0644)
}

// NamedGenome pairs a display name (derived from the filename) with a Genome.
type NamedGenome struct {
	Name   string
	Genome *Genome
}

// sanitizeFilename converts a user-supplied name into a safe filename component.
func sanitizeFilename(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '-', r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('_')
		}
	}
	s := b.String()
	if len(s) > 64 {
		s = s[:64]
	}
	return s
}

// SaveCreatureToFileNamed saves a genome with a user-provided name as the filename.
// If name is empty or produces no safe characters, falls back to timestamp-based naming.
// Appends a timestamp suffix if a file with that name already exists.
func SaveCreatureToFileNamed(g *Genome, name string) error {
	if err := os.MkdirAll(creaturesSaveDir, 0755); err != nil {
		return err
	}
	safe := sanitizeFilename(name)
	var filename string
	if safe == "" {
		filename = fmt.Sprintf("%d_%d.json", time.Now().UnixNano(), rand.Int63())
	} else {
		filename = safe + ".json"
		if _, err := os.Stat(filepath.Join(creaturesSaveDir, filename)); err == nil {
			filename = fmt.Sprintf("%s_%d.json", safe, time.Now().UnixNano())
		}
	}
	data := toGenomeData(g)
	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(creaturesSaveDir, filename), encoded, 0644)
}

// LoadAllCreatureGenomesNamed reads all saved genome files and returns them with
// display names derived from their filenames (extension stripped).
func LoadAllCreatureGenomesNamed() ([]NamedGenome, error) {
	entries, err := os.ReadDir(creaturesSaveDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var result []NamedGenome
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		raw, readErr := os.ReadFile(filepath.Join(creaturesSaveDir, entry.Name()))
		if readErr != nil {
			continue
		}
		var gd genomeData
		if jsonErr := json.Unmarshal(raw, &gd); jsonErr != nil {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".json")
		result = append(result, NamedGenome{Name: name, Genome: fromGenomeData(gd)})
	}
	return result, nil
}

// LoadAllCreatureGenomes reads all individually saved creature genome files from data/creatures/.
// Returns (nil, nil) if the directory does not exist.
func LoadAllCreatureGenomes() ([]*Genome, error) {
	entries, err := os.ReadDir(creaturesSaveDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var genomes []*Genome
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(creaturesSaveDir, entry.Name()))
		if err != nil {
			continue
		}
		var gd genomeData
		if err := json.Unmarshal(raw, &gd); err != nil {
			continue
		}
		genomes = append(genomes, fromGenomeData(gd))
	}
	return genomes, nil
}

// LoadGenomesFromFile reads genomes from a JSON file.
// Returns (nil, nil) if the file does not exist.
func LoadGenomesFromFile(path string) ([]*Genome, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var data []genomeData
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	genomes := make([]*Genome, len(data))
	for i, gd := range data {
		genomes[i] = fromGenomeData(gd)
	}
	return genomes, nil
}
