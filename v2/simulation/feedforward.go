package simulation

import (
	"biogo/v2/grid"
	"fmt"
	"math"
)

func (c *Creature) FeedForward(w *grid.World, p *Population, step int, params *Parameters) []float32 {
	actionLevels := make([]float32, ACTION_COUNT)
	neuronAccumulators := map[byte]float32{}
	neuronOutputsEvaluated := false
	const decayRate = 0.0005 // Tiny constant decay
	const learningRate = 0.01
	const learningThreshold = 0.1
	const energyCostOfLearning = 0.005

	for i, gene := range c.Nnet.Edges {
		instinctWeight := gene.WeightAsFloat32()
		if c.Nnet.Weights[i] > instinctWeight {
			c.Nnet.Weights[i] -= decayRate
		} else if c.Nnet.Weights[i] < instinctWeight {
			c.Nnet.Weights[i] += decayRate
		}

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
			inputVal = c.GetSensor(gene.SourceID, w, p, step, params)
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

		currentWeight := c.Nnet.Weights[i]
		isAction := gene.SinkType == ACTION
		isNeuron := gene.SinkType == NEURON

		if isAction {
			actionLevels[gene.SinkID] += inputVal * currentWeight
		} else {
			neuronAccumulators[gene.SinkID] += inputVal * currentWeight
		}

		if isNeuron || (isAction && len(c.Nnet.HiddenNeurons) == 0) {
			var sinkOutput float32
			if isNeuron {
				sinkOutput = c.Nnet.HiddenNeurons[gene.SinkID].Output
			} else {
				// For Actions, we use the current accumulated signal as the "output"
				sinkOutput = actionLevels[gene.SinkID]
			}
			correlation := inputVal * sinkOutput

			energyThreshold := float32(c.Genome.MaxEnergy) * 0.6

			if c.Energy > energyThreshold && c.Dopamine > 0.1 {

				learningSignal := correlation * c.Dopamine

				if learningSignal > learningThreshold {
					// 3. Apply the update
					c.Nnet.Weights[i] += learningRate * learningSignal

					// 4. Metabolic Tax
					c.Energy -= energyCostOfLearning

					// 5. Clamp Weight
					if c.Nnet.Weights[i] > 4.0 {
						c.Nnet.Weights[i] = 4.0
					} else if c.Nnet.Weights[i] < -4.0 {
						c.Nnet.Weights[i] = -4.0
					}
				}
			}
		}
	}
	return actionLevels
}
