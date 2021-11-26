package simulation

import (
	"fmt"
	"gopop/v2/grid"
	"math"
)

// The "Brains"

func (c *Creature) FeedForward(g *grid.Grid, p *Population, step int) []float32 {
	actionLevels := make([]float32, ACTION_COUNT)
	neuronAccumulators := map[byte]float32{}
	neuronOutputsEvaluated := false

	for _, gene := range c.Nnet.Edges {

		// First we evaluate the outputs to ACTIONS
		if gene.SinkType == ACTION && !neuronOutputsEvaluated {
			for key, neuron := range c.Nnet.HiddenNeurons {
				if neuron.Driven {
					neuron.Output = float32(math.Tanh(float64(neuronAccumulators[key])))
				}
			}
			neuronOutputsEvaluated = true
		}

		var inputVal float32
		if gene.SourceType == SENSOR {
			inputVal = c.GetSensor(gene.SourceID, g, p, step)
		} else {
			if _, ok := c.Nnet.HiddenNeurons[gene.SourceID]; !ok {
				fmt.Printf("\n\nNot okay, trying to see %d of type %d, %s", gene.SourceID, gene.SourceType, c.Nnet.String())
				for _, gene := range c.Nnet.Edges {
					fmt.Printf("\n%s", gene.PrettyString())
				}
				fmt.Printf("\nC.Nnet.HiddenNeurons: %v\n", c.Nnet.HiddenNeurons)
			}
			inputVal = c.Nnet.HiddenNeurons[gene.SourceID].Output
		}

		if gene.SinkType == ACTION {
			actionLevels[gene.SinkID] += inputVal * gene.WeightAsFloat32()
		} else {
			neuronAccumulators[gene.SinkID] += inputVal * gene.WeightAsFloat32()
		}
	}
	return actionLevels
}
