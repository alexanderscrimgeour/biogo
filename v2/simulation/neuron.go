package simulation

import (
	"fmt"
	"sort"
)

const (
	// Neurons are treated differently to sensors/actions.
	NEURON = iota
	SENSOR
	ACTION
)

type Neuron struct {
	Output float32
	Driven bool
}

type NeuralNet struct {
	Edges            []Gene
	HiddenNeurons    [256]*Neuron       // indexed by neuron ID; sparse, use HiddenNeuronIDs to iterate
	HiddenNeuronIDs  []byte             // sorted list of occupied indices
	ActiveSensors    [SENSOR_COUNT]bool // true for each sensor ID wired into at least one edge; set once at construction
	Weights          []float32
	LastSensorValues [SENSOR_COUNT]float32
	LastActionValues []float32
}

type Node struct {
	NewID         byte // New ID post removal of useless neurons
	OutputCount   byte // Outputs
	InputCount    byte // InputCount does not include self-loops
	SelfLoopCount byte // Self loop inputs
}

// A NodeMap is used to store the hidden layer NEURONS in a graph, such that we're able to
// Determine which are "useless" neurons and remove them before constructing the nnet
type NodeMap map[byte]*Node

func (n Neuron) String() string {
	return fmt.Sprintf("[Output: %f, Driven: %t],", n.Output, n.Driven)
}

func (n NeuralNet) String() string {
	str := "NNET| Edges: ["
	for _, val := range n.Edges {
		str += fmt.Sprintf("%s ", val.String())
	}
	str += "]\n    | Neurons: "
	for _, id := range n.HiddenNeuronIDs {
		if n.HiddenNeurons[id] != nil {
			str += fmt.Sprintf("%s ", n.HiddenNeurons[id].String())
		}
	}
	str += "\n"
	return str
}

func (n Node) String() string {
	return fmt.Sprintf("%d:{output: %d,input: %d, selfLoops: %d}", n.NewID, n.OutputCount, n.InputCount, n.SelfLoopCount)
}

func (n NodeMap) String() string {
	str := "["
	for key, val := range n {
		str += fmt.Sprintf("%d->%s\n, ", key, val)
	}
	str += "]"
	return str
}

func CreateInitialNeuronOutput() float32 { return 0.5 }

func CreateNeuralNetworkFromGenome(genes []Gene, CognitiveBreadth byte) *NeuralNet {
	neuralGenes := convertGenesToNeuronIDs(genes, CognitiveBreadth)
	nodeMap := createNodeMap(neuralGenes)
	finalGenes := removeUselessGenes(neuralGenes, nodeMap)
	// The remaining nodes in nodeMap will need to be re-indexed
	setNodeNewIDValues(nodeMap)
	neuralNet := createNeuralNetworkFromGenesAndNodeMap(finalGenes, nodeMap)
	return neuralNet
}

func createNeuralNetworkFromGenesAndNodeMap(g []Gene, n NodeMap) *NeuralNet {
	nnet := NeuralNet{}
	nnet.Weights = make([]float32, len(g))

	// Two-pass: neuron-sink edges first (improves feedforward locality), then action-sink.
	edgeIndex := 0
	for _, gene := range g {
		if gene.SinkType == NEURON {
			gene.SinkID = n[gene.SinkID].NewID
			if gene.SourceType == NEURON {
				gene.SourceID = n[gene.SourceID].NewID
			}
			nnet.Edges = append(nnet.Edges, gene)
			nnet.Weights[edgeIndex] = gene.WeightAsFloat32()
			edgeIndex++
		}
	}
	for _, gene := range g {
		if gene.SinkType == ACTION {
			if gene.SourceType == NEURON {
				gene.SourceID = n[gene.SourceID].NewID
			}
			nnet.Edges = append(nnet.Edges, gene)
			nnet.Weights[edgeIndex] = gene.WeightAsFloat32()
			edgeIndex++
		}
	}
	// Create the neurons; HiddenNeuronIDs is sorted for deterministic iteration.
	nnet.HiddenNeuronIDs = make([]byte, 0, len(n))
	for _, node := range n {
		neuron := &Neuron{
			Output: CreateInitialNeuronOutput(),
			Driven: node.InputCount != 0,
		}
		nnet.HiddenNeurons[node.NewID] = neuron
		nnet.HiddenNeuronIDs = append(nnet.HiddenNeuronIDs, node.NewID)
	}
	sort.Slice(nnet.HiddenNeuronIDs, func(i, j int) bool {
		return nnet.HiddenNeuronIDs[i] < nnet.HiddenNeuronIDs[j]
	})

	// Record which sensors are actually wired in so FeedForward can pre-compute
	// them once rather than calling GetSensor once per edge.
	for i := range nnet.Edges {
		if nnet.Edges[i].SourceType == SENSOR {
			nnet.ActiveSensors[nnet.Edges[i].SourceID] = true
		}
	}

	return &nnet
}

func setNodeNewIDValues(n NodeMap) {
	keys := make([]byte, 0, len(n))
	for k := range n {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	for i, key := range keys {
		node := n[key]
		if node.OutputCount == 0 {
			panic("Somehow we have a dead neuron!")
		}
		node.NewID = byte(i)
	}
}
func removeUselessGenes(g []Gene, n NodeMap) []Gene {
	if len(n) == 0 {
		return g
	}

	final := g
	done := false
	for !done {
		done = true

		// 1. Extract and sort keys to ensure deterministic pruning
		keys := make([]byte, 0, len(n))
		for k := range n {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			return keys[i] < keys[j]
		})

		// 2. Iterate through sorted keys
		for _, key := range keys {
			node, exists := n[key]
			if !exists {
				continue
			}

			// If a neuron only outputs to itself or nothing, it's useless
			if node.OutputCount == node.SelfLoopCount {
				done = false
				final = removeConnectionsToGene(final, n, key)
				delete(n, key)
			}
		}
	}
	return final
}
func removeConnectionsToGene(genes []Gene, n NodeMap, key byte) []Gene {
	newGenes := genes[:0]

	for _, gene := range genes {
		// If this gene's destination (Sink) is the neuron we are deleting...
		if gene.SinkType == NEURON && gene.SinkID == key {
			// ...then the source of this gene just lost an output.
			if gene.SourceType == NEURON {
				if sourceNode, exists := n[gene.SourceID]; exists {
					if sourceNode.OutputCount > 0 {
						sourceNode.OutputCount--
					}
					// Handle self-loop decrement if the neuron is pruning itself
					if gene.SourceID == key && sourceNode.SelfLoopCount > 0 {
						sourceNode.SelfLoopCount--
					}
				}
			}
			// Do NOT append this gene; it is now deleted.
			continue
		}

		// Keep genes that don't point to the deleted neuron.
		newGenes = append(newGenes, gene)
	}
	return newGenes
}

func TestRemove(nodeMap NodeMap, key byte) {
	delete(nodeMap, key)
}

func TestRemoveList(g []Gene) []Gene {
	g = g[:len(g)-1]
	return g
}

// CreateNodeMap takes in
func createNodeMap(neuralGenes []Gene) NodeMap {
	nMap := NodeMap{}
	ensureNode := func(id byte) {
		if _, ok := nMap[id]; !ok {
			nMap[id] = &Node{
				NewID:         0,
				OutputCount:   0,
				InputCount:    0,
				SelfLoopCount: 0,
			}
		}
	}

	for _, gene := range neuralGenes {
		// First, if this gene -> NEURON, it must be from a SENSOR or a NEURON
		if gene.SinkType == NEURON {
			ensureNode(gene.SinkID)
			if gene.SourceType == NEURON && gene.SourceID == gene.SinkID {
				nMap[gene.SinkID].SelfLoopCount++
			} else {
				// We count inputs from SENSORS or other NEURONS
				nMap[gene.SinkID].InputCount++
			}
		}

		// If NEURON -> GENE, can only be a NEURON or an ACTION
		if gene.SourceType == NEURON {
			ensureNode(gene.SourceID)
			nMap[gene.SourceID].OutputCount++
		}
	}
	return nMap
}

func convertGenesToNeuronIDs(genes []Gene, CognitiveBreadth byte) []Gene {
	newGenes := make([]Gene, len(genes))
	for i, g := range genes {
		if g.SourceType == NEURON && CognitiveBreadth > 0 {
			g.SourceID %= CognitiveBreadth
		} else {
			g.SourceType = SENSOR
			g.SourceID %= SENSOR_COUNT
		}
		if g.SinkType == NEURON && CognitiveBreadth > 0 {
			g.SinkID %= CognitiveBreadth
		} else {
			g.SinkType = ACTION
			g.SinkID %= ACTION_COUNT
		}
		newGenes[i] = g
	}
	return newGenes
}
