package simulation

import (
	"biogo/v2/grid"
	"fmt"
	"math"
)

func (c *Creature) FeedForward(w *grid.World, p *Population, step int, params *Parameters) []float32 {
	actionLevels := make([]float32, ACTION_COUNT)
	var neuronAccumulators [256]float32 // stack-allocated; no heap allocation per tick
	neuronOutputsEvaluated := false
	// Reuse the sensor-values map across ticks to avoid per-tick allocation.
	if c.Nnet.LastSensorValues == nil {
		c.Nnet.LastSensorValues = make(map[byte]float32)
	} else {
		clear(c.Nnet.LastSensorValues)
	}
	const decayRate = 0.0005
	const energyCostOfLearning = 0.005

	genomeLearningRate := params.MinLearningRate + float32(c.Genome.LearningRate)/255.0*(params.MaxLearningRate-params.MinLearningRate)
	learningThreshold := params.MinLearningThreshold + float32(c.Genome.LearningThreshold)/255.0*(params.MaxLearningThreshold-params.MinLearningThreshold)

	// Modulate learning rate from previous tick's SET_LEARNING_RATE output; tanh maps to [-1, 1] so rate scales in [0, 2x].
	var learningRateMod float32
	if len(c.Nnet.LastActionValues) > int(SET_LEARNING_RATE) {
		learningRateMod = float32(math.Tanh(float64(c.Nnet.LastActionValues[SET_LEARNING_RATE])))
	}
	learningRate := genomeLearningRate * (1 + learningRateMod)
	if learningRate < 0 {
		learningRate = 0
	}

	for i, gene := range c.Nnet.Edges {
		instinctWeight := gene.WeightAsFloat32()
		if c.Nnet.Weights[i] > instinctWeight {
			c.Nnet.Weights[i] -= decayRate
		} else if c.Nnet.Weights[i] < instinctWeight {
			c.Nnet.Weights[i] += decayRate
		}

		if gene.SinkType == ACTION && !neuronOutputsEvaluated {
			for _, key := range c.Nnet.HiddenNeuronIDs {
				neuron := c.Nnet.HiddenNeurons[key]
				if neuron.Driven {
					neuron.Output = float32(math.Tanh(float64(neuronAccumulators[key])))
				}
			}
			neuronOutputsEvaluated = true
		}

		var inputVal float32
		if gene.SourceType == SENSOR {
			inputVal = c.GetSensor(gene.SourceID, w, p, step, params)
			c.Nnet.LastSensorValues[gene.SourceID] = inputVal
		} else {
			if c.Nnet.HiddenNeurons[gene.SourceID] == nil {
				fmt.Printf("\n\nNot okay, trying to see %d of type %d, %s", gene.SourceID, gene.SourceType, c.Nnet.String())
				for _, gene := range c.Nnet.Edges {
					fmt.Printf("\n%s", gene.PrettyString())
				}
				fmt.Printf("\nC.Nnet.HiddenNeuronIDs: %v\n", c.Nnet.HiddenNeuronIDs)
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

		if isNeuron || (isAction && len(c.Nnet.HiddenNeuronIDs) == 0) {
			var sinkOutput float32
			if isNeuron {
				sinkOutput = c.Nnet.HiddenNeurons[gene.SinkID].Output
			} else {
				// For Actions, we use the current accumulated signal as the "output"
				sinkOutput = actionLevels[gene.SinkID]
			}
			correlation := inputVal * sinkOutput

			energyThreshold := c.MaxEnergy(params) * 0.6

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
	c.Nnet.LastActionValues = actionLevels
	return actionLevels
}
