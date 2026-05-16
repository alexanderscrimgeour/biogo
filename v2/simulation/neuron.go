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
	Output        float32
	Driven        bool
	AverageOutput float32
	Sensitivity   float32
}

type NeuralNet struct {
	Edges            []Gene
	HiddenNeurons    []Neuron           // packed slice, indexed 0..N-1 by NewID; iterate directly
	ActiveSensors    [SENSOR_COUNT]bool // true for each sensor ID wired into at least one edge; set once at construction
	ActiveActions    [ACTION_COUNT]bool // true for each action ID that is a sink in at least one edge; set once at construction
	NeedsKinship     bool               // true if any kinship sensor (KINSHIP_LOCAL/NEAREST/NEAREST_DISTANCE) is wired in
	Weights          []float32
	LastSensorValues [SENSOR_COUNT]float32
	LastActionValues [ACTION_COUNT]float32
	NeuronEdgeCount  int // number of leading Edges with SinkType==NEURON (all come before ACTION edges)
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
	for _, neuron := range n.HiddenNeurons {
		str += fmt.Sprintf("%s ", neuron.String())
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

func CreateNeuralNetworkFromGenome(genes []Gene, cognitiveBreadth byte) *NeuralNet {
	neuralGenes := convertGenesToNeuronIDs(genes, cognitiveBreadth)
	nodeMap := createNodeMap(neuralGenes)
	finalGenes := removeUselessGenes(neuralGenes, nodeMap)
	// The remaining nodes in nodeMap will need to be re-indexed
	setNodeNewIDValues(nodeMap)
	neuralNet := createNeuralNetworkFromGenesAndNodeMap(finalGenes, nodeMap, cognitiveBreadth)
	return neuralNet
}

func createNeuralNetworkFromGenesAndNodeMap(g []Gene, n NodeMap, cognitiveBreadth byte) *NeuralNet {
	nnet := NeuralNet{}
	nnet.Weights = make([]float32, len(g))

	allowedSensorCount := getAllowedSensorCount(cognitiveBreadth)
	allowedActionCount := getAllowedActionCount(cognitiveBreadth)

	// Two-pass: neuron-sink edges first, then action-sink.
	edgeIndex := 0
	for _, gene := range g {
		if gene.SinkType == NEURON {
			// Skip pruned in nMap
			node, ok := n[gene.SinkID]
			if !ok {
				continue
			}
			gene.SinkID = node.NewID

			if gene.SourceType == NEURON {
				if srcNode, srcOk := n[gene.SourceID]; srcOk {
					gene.SourceID = srcNode.NewID
				} else {
					continue // Prune if driving neuron doesn't exist
				}
			} else {
				// Clamp sensor access to currently allowed tier bounds
				gene.SourceID %= allowedSensorCount
			}

			nnet.Edges = append(nnet.Edges, gene)
			nnet.Weights[edgeIndex] = gene.WeightAsFloat32()
			edgeIndex++
		}
	}
	nnet.NeuronEdgeCount = edgeIndex
	for _, gene := range g {
		if gene.SinkType == ACTION {
			// Clamp target actions to currently allowed tier bounds
			gene.SinkID %= allowedActionCount

			if gene.SourceType == NEURON {
				if srcNode, srcOk := n[gene.SourceID]; srcOk {
					gene.SourceID = srcNode.NewID
				} else {
					continue // Prune if driving neuron doesn't exist
				}
			} else {
				// Clamp sensor access to currently allowed tier bounds
				gene.SourceID %= allowedSensorCount
			}

			nnet.Edges = append(nnet.Edges, gene)
			nnet.Weights[edgeIndex] = gene.WeightAsFloat32()
			edgeIndex++
		}
	}
	// Create the neurons. NewIDs are assigned 0..N-1 by setNodeNewIDValues, so the
	// slice is fully packed and edge SourceID/SinkID values remain valid indices.
	nnet.HiddenNeurons = make([]Neuron, len(n))
	for _, node := range n {
		nnet.HiddenNeurons[node.NewID] = Neuron{
			Output:        CreateInitialNeuronOutput(),
			Driven:        node.InputCount != 0,
			AverageOutput: CreateInitialNeuronOutput(), // Matches your updated struct
			Sensitivity:   1.0,
		}
	}
	// Record which sensors and actions are actually wired in.
	for i := range nnet.Edges {
		if nnet.Edges[i].SourceType == SENSOR {
			nnet.ActiveSensors[nnet.Edges[i].SourceID] = true
		}
		if nnet.Edges[i].SinkType == ACTION {
			nnet.ActiveActions[nnet.Edges[i].SinkID] = true
		}
	}
	nnet.NeedsKinship = nnet.ActiveSensors[KINSHIP_LOCAL] ||
		nnet.ActiveSensors[KINSHIP_NEAREST] ||
		nnet.ActiveSensors[KINSHIP_NEAREST_DISTANCE]
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

// removeUselessGenes iteratively purges non-functional neurons and dead connections from the network.
func removeUselessGenes(g []Gene, n NodeMap) []Gene {
	if len(n) == 0 {
		return g
	}

	final := g
	done := false
	for !done {
		done = true

		// Extract and sort keys to ensure deterministic pruning
		keys := make([]byte, 0, len(n))
		for k := range n {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			return keys[i] < keys[j]
		})

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
			} else if node.InputCount == 0 && node.SelfLoopCount == 0 {
				// No incoming connections: outputs a constant, carries no sensor signal
				done = false
				final = removeOutgoingConnections(final, n, key)
				delete(n, key)
			}
		}
	}
	return final
}

// removeConnectionsToGene removes all edges that sink (target) into the specified neuron key.
// It simultaneously updates the OutputCount metrics of any upstream hidden neurons that were driving it.
func removeConnectionsToGene(genes []Gene, n NodeMap, key byte) []Gene {
	newGenes := genes[:0]

	for _, gene := range genes {
		// If this gene's destination (Sink) is the neuron we are deleting...
		if gene.SinkType == NEURON && gene.SinkID == key {
			// ...then the source of this gene just lost an output.
			if gene.SourceType == NEURON {
				if gene.SourceID != key {
					// Handle self-loop decrement if the neuron is pruning itself
					if sourceNode, exists := n[gene.SourceID]; exists {
						if sourceNode.OutputCount > 0 {
							sourceNode.OutputCount--
						}
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

// removeOutgoingConnections removes all edges that source (originate) from the specified neuron key.
// It simultaneously updates the InputCount metrics of any downstream hidden neurons receiving its signal.
func removeOutgoingConnections(genes []Gene, n NodeMap, key byte) []Gene {
	newGenes := genes[:0]

	for _, gene := range genes {
		if gene.SourceType == NEURON && gene.SourceID == key {
			if gene.SinkType == NEURON {
				if gene.SinkID != key {
					if sinkNode, exists := n[gene.SinkID]; exists {
						if sinkNode.InputCount > 0 {
							sinkNode.InputCount--
						}
					}
				}
			}
			continue
		}
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

// createNodeMap analyzes the topology of the neural network by scanning all connections (edges).
// It constructs a map tracking the structural characteristics of every unique hidden neuron found.
//
// This analysis is a prerequisite for network optimization:
//  1. It identifies non-functional "dead-end" neurons (e.g., hidden neurons that receive sensor
//     inputs but never output to anything, or neurons that output to actions but have no inputs).
//  2. It separates true independent inputs from internal self-recurrent loops, allowing
//     the system to track if a neuron is actively "driven" by external signals.
//  3. It collects the raw structural layout needed to compact the surviving neuron IDs
//     into a packed, contiguous memory space (0..N-1) for fast execution lookups.
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

func convertGenesToNeuronIDs(genes []Gene, cognitiveBreadth byte) []Gene {
	newGenes := make([]Gene, len(genes))

	allowedSensorCount := getAllowedSensorCount(cognitiveBreadth)
	allowedActionCount := getAllowedActionCount(cognitiveBreadth)

	for i, g := range genes {
		if g.SourceType == NEURON {
			if cognitiveBreadth > 0 {
				g.SourceID %= cognitiveBreadth
			} else {
				// Creature has no capacity for hidden neurons.
				// Set ID to 0; createNodeMap will naturally prune this useless edge out.
				g.SourceID = 0
			}
		} else {
			// SENSOR
			g.SourceID %= allowedSensorCount
		}

		if g.SinkType == NEURON {
			if cognitiveBreadth > 0 {
				g.SinkID %= cognitiveBreadth
			} else {
				// No hidden neuron capacity.
				g.SinkID = 0
			}
		} else {
			// ACTION
			g.SinkID %= allowedActionCount
		}

		newGenes[i] = g
	}
	return newGenes
}

func getAllowedSensorCount(breadth byte) byte {
	if breadth < 64 {
		return MaxTier1Sensor + 1
	}
	if breadth < 128 {
		return MaxTier2Sensor + 1
	}
	if breadth < 192 {
		return MaxTier3Sensor + 1
	}
	return SENSOR_COUNT
}

func getAllowedActionCount(breadth byte) byte {
	if breadth < 64 {
		return MaxTier1Action + 1
	}
	if breadth < 128 {
		return MaxTier2Action + 1
	}
	if breadth < 192 {
		return MaxTier3Action + 1
	}
	return ACTION_COUNT
}
