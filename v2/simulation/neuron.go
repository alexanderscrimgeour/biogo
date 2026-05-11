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
	Edges            []*Gene
	HiddenNeurons    [256]*Neuron // indexed by neuron ID; sparse, use HiddenNeuronIDs to iterate
	HiddenNeuronIDs  []byte       // sorted list of occupied indices
	Weights          []float32
	LastSensorValues map[byte]float32
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

func CreateNeuralNetworkFromGenome(genes []*Gene, neuronCount byte) *NeuralNet {
	neuralGenes := convertGenesToNeuronIDs(genes, neuronCount)
	nodeMap := createNodeMap(neuralGenes)
	finalGenes := removeUselessGenes(neuralGenes, nodeMap)
	// The remaining nodes in nodeMap will need to be re-indexed
	setNodeNewIDValues(nodeMap)
	neuralNet := createNeuralNetworkFromGenesAndNodeMap(finalGenes, nodeMap)
	return neuralNet
}

func createNeuralNetworkFromGenesAndNodeMap(g []*Gene, n NodeMap) *NeuralNet {
	nnet := NeuralNet{}
	nnet.Weights = make([]float32, len(g))

	// We do this in two phases, first we add the -> neurons, then we add the -> actions. This
	// Improves the performance of the fforward
	edgeIndex := 0
	for _, gene := range g {
		if gene.SinkType == NEURON {
			// Create gene copy
			new := *gene

			// Fix the Sink id
			new.SinkID = n[gene.SinkID].NewID
			// If we're coming _from_ a NEURON, we need to fix it too
			if gene.SourceType == NEURON {
				new.SourceID = n[gene.SourceID].NewID
			}
			// Add the new gene to the nnet
			nnet.Edges = append(nnet.Edges, &new)
			// Initialise weights
			nnet.Weights[edgeIndex] = new.WeightAsFloat32()
			edgeIndex++
		}
	}
	for _, gene := range g {
		if gene.SinkType == ACTION {
			new := *gene
			if gene.SourceType == NEURON {
				new.SourceID = n[gene.SourceID].NewID
			}
			nnet.Edges = append(nnet.Edges, &new)
			nnet.Weights[edgeIndex] = new.WeightAsFloat32()
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
func removeUselessGenes(g []*Gene, n NodeMap) []*Gene {
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
func removeConnectionsToGene(genes []*Gene, n NodeMap, key byte) []*Gene {
	newGenes := make([]*Gene, 0, len(genes))

	for _, gene := range genes {
		if gene.SinkType == NEURON && gene.SinkID == key {

			if gene.SourceType == NEURON {
				sourceNode, exists := n[gene.SourceID]
				if exists {
					if sourceNode.OutputCount > 0 {
						sourceNode.OutputCount--
					}
					if gene.SourceID == key && sourceNode.SelfLoopCount > 0 {
						sourceNode.SelfLoopCount--
					}
				}
			}
		} else {
			newGenes = append(newGenes, gene)
		}
	}
	return newGenes
}

func TestRemove(nodeMap NodeMap, key byte) {
	delete(nodeMap, key)
}

func TestRemoveList(g []*Gene) []*Gene {
	g = g[:len(g)-1]
	return g
}

// CreateNodeMap takes in
func createNodeMap(neuralGenes []*Gene) NodeMap {
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

func convertGenesToNeuronIDs(genes []*Gene, neuronCount byte) []*Gene {
	newGenes := make([]*Gene, len(genes))

	for i, gene := range genes {
		new := *gene // Make a copy
		if new.SourceType == NEURON && neuronCount > 0 {
			// treat neurons as neurons ONLY if there's a neuronCount > 0
			new.SourceID %= neuronCount
		} else {
			// Reset the type in case of Neurons with neuronCount == 0
			new.SourceType = SENSOR
			new.SourceID %= SENSOR_COUNT
		}

		if new.SinkType == NEURON && neuronCount > 0 {
			// treat neurons as neurons ONLY if there's a neuronCount > 0
			new.SinkID %= neuronCount
		} else {
			// Reset the type in case of Neurons with neuronCount == 0
			new.SinkType = ACTION
			new.SinkID %= ACTION_COUNT
		}
		newGenes[i] = &new
	}
	return newGenes
}
