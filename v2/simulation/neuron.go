package simulation

import "fmt"

const (
	// Neurons are treated differently to sensors/actions.
	NEURON = 0
	SENSOR = 1
	ACTION = 1
)

type Neuron struct {
	Output float32
	Driven bool
}

type NeuralNet struct {
	Edges         []*Gene
	HiddenNeurons map[byte]*Neuron
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
	for _, val := range n.HiddenNeurons {
		str += fmt.Sprintf("%s ", val.String())
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

	// We do this in two phases, first we add the -> neurons, then we add the -> actions. This
	// Improves the performance of the fforward
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
		}
	}
	for _, gene := range g {
		if gene.SinkType == ACTION {
			new := *gene
			if gene.SourceType == NEURON {
				new.SourceID = n[gene.SourceID].NewID
			}
			nnet.Edges = append(nnet.Edges, &new)
		}

	}
	nnet.HiddenNeurons = make(map[byte]*Neuron, len(n))
	// Create the neurons
	for id, node := range n {
		neuron := &Neuron{
			Output: CreateInitialNeuronOutput(),
			Driven: node.InputCount != 0,
		}
		nnet.HiddenNeurons[id] = neuron
	}
	return &nnet
}

func setNodeNewIDValues(n NodeMap) {
	i := 0
	for _, node := range n {
		if node.OutputCount == 0 {
			panic("Somehow we have a dead neuron!")
		}
		node.NewID = byte(i)
		i++
	}
}

func removeUselessGenes(g []*Gene, n NodeMap) []*Gene {
	// Do not cull if no neurons
	if len(n) == 0 {
		return g
	}
	final := g
	// Iterate until we're done
	done := false
	for !done {
		done = true
		for key, node := range n {
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
	new := []*Gene{}
	for _, gene := range genes {
		if gene.SinkType == NEURON && gene.SinkID == key {
			if gene.SourceType == NEURON {
				if n[gene.SourceID].OutputCount >= 0 {
					n[gene.SourceID].OutputCount--
				}
			}
		} else {
			new = append(new, gene)
		}
	}
	return new
}

func TestRemove(nodeMap NodeMap, key byte) {
	delete(nodeMap, key)
}

func TestRemoveList(g []*Gene) []*Gene {
	g = g[:len(g)-1]
	return g
}

// CreateNodeMap takes in a
func createNodeMap(neuralGenes []*Gene) NodeMap {
	nMap := NodeMap{}
	for _, gene := range neuralGenes {
		// First, if this gene -> NEURON, it must be from a SENSOR or a NEURON
		if gene.SinkType == NEURON {
			// New Node
			node := &Node{
				NewID:         0,
				OutputCount:   0,
				InputCount:    0,
				SelfLoopCount: 0,
			}
			// If not present, add new node to the nMap
			if _, ok := nMap[gene.SinkID]; !ok {
				nMap[gene.SinkID] = node
			}
			// If Neuron, we allow self inputs, so increment self loop count
			if (gene.SourceType == NEURON) && (gene.SourceID == gene.SinkID) {
				nMap[gene.SinkID].SelfLoopCount++
			} else {
				// Case SENSOR we just increment inputs
				nMap[gene.SinkID].InputCount++
			}
		}

		// If NEURON -> GENE, can only be a NEURON or an ACTION
		if gene.SourceType == NEURON {
			// New node object
			node := &Node{
				NewID:         0,
				OutputCount:   0,
				InputCount:    0,
				SelfLoopCount: 0,
			}
			// If not present, add new node to the nMap
			if _, ok := nMap[gene.SourceID]; !ok {
				nMap[gene.SourceID] = node
			}
			// Add the output count
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
			new.SourceType = 1
			new.SourceID %= SENSOR_COUNT
		}

		if new.SinkType == NEURON && neuronCount > 0 {
			// treat neurons as neurons ONLY if there's a neuronCount > 0
			new.SinkID %= neuronCount
		} else {
			// Reset the type in case of Neurons with neuronCount == 0
			new.SinkType = 1
			new.SinkID %= ACTION_COUNT
		}
		newGenes[i] = &new
	}
	return newGenes
}
