package simulation

import (
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
