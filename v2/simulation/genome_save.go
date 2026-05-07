package simulation

import (
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
)

// MaxSavedGenomes is the number of representative genomes persisted by SaveBestGenomes.
const MaxSavedGenomes = 5

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
	MaxEnergy        byte       `json:"max_energy"`
	SightDistance    byte       `json:"sight_distance"`
	FieldOfView      byte       `json:"field_of_view"`
	Responsiveness   byte       `json:"responsiveness"`
	MutationRate     byte       `json:"mutation_rate"`
	Size             byte       `json:"size"`
	MinSize          byte       `json:"min_size"`
	ReproductionType byte       `json:"reproduction_type"`
	NeuronCount      byte       `json:"neuron_count"`
	BrainLength      byte       `json:"brain_length"`
	JuvenilePeriod   byte       `json:"juvenile_period"`
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
		MaxEnergy:        g.MaxEnergy,
		SightDistance:    g.SightDistance,
		FieldOfView:      g.FieldOfView,
		Responsiveness:   g.Responsiveness,
		MutationRate:     g.MutationRate,
		Size:             g.Size,
		MinSize:          g.MinSize,
		ReproductionType: g.ReproductionType,
		NeuronCount:      g.NeuronCount,
		BrainLength:      g.BrainLength,
		JuvenilePeriod:   g.JuvenilePeriod,
		Brain:            genes,
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
	return &Genome{
		OscPeriod:        gd.OscPeriod,
		MaxEnergy:        gd.MaxEnergy,
		SightDistance:    gd.SightDistance,
		FieldOfView:      gd.FieldOfView,
		Responsiveness:   gd.Responsiveness,
		MutationRate:     gd.MutationRate,
		Size:             gd.Size,
		MinSize:          gd.MinSize,
		ReproductionType: gd.ReproductionType,
		NeuronCount:      gd.NeuronCount,
		BrainLength:      gd.BrainLength,
		JuvenilePeriod:   gd.JuvenilePeriod,
		Brain:            genes,
	}
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
			if sim := GenomeSimilarity(*c.Genome, *cl.genome); sim > bestSim {
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
