package simulation

import (
	"biogo/v2/world"
	"math"
)

const energyCostOfFiring = 0.0001
const decayRate = 0.0005
const energyCostOfLearning = 0.005

func (c *Creature) FeedForward(w *world.World, p *Population, step int, params *Parameters) []float32 {
	var neuroplasticityMod float32
	if len(c.Nnet.LastActionValues) > int(SET_LEARNING_RATE) {
		neuroplasticityMod = softsign(c.Nnet.LastActionValues[SET_LEARNING_RATE])
	}

	if len(c.Nnet.LastActionValues) != int(ACTION_COUNT) {
		c.Nnet.LastActionValues = make([]float32, ACTION_COUNT)
	} else {
		clear(c.Nnet.LastActionValues)
	}
	actionLevels := c.Nnet.LastActionValues

	var neuronAccumulators [256]float32
	neuronOutputsEvaluated := false

	// update the SensorContext once to eliminate duplicate spatial queries
	c.UpdateSensorContext(w, p, params)
	ctx := &c.Sensors

	var sensorCache [SENSOR_COUNT]float32
	c.Nnet.LastSensorValues = [SENSOR_COUNT]float32{}
	for sid := byte(0); sid < SENSOR_COUNT; sid++ {
		if c.Nnet.ActiveSensors[sid] {
			// Pass the pre-built context to GetSensor
			val := c.GetSensor(sid, w, p, ctx, step, params)
			sensorCache[sid] = val
			c.Nnet.LastSensorValues[sid] = val
		}
	}

	genomeNeuroplasticity := params.MinNeuroplasticity + float32(c.Genome.Neuroplasticity)/255.0*(params.MaxNeuroplasticity-params.MinNeuroplasticity)
	learningThreshold := params.MinLearningThreshold + float32(c.Genome.LearningThreshold)/255.0*(params.MaxLearningThreshold-params.MinLearningThreshold)

	dopamineDelta := c.Dopamine - c.LastDopamine
	absDelta := dopamineDelta
	if absDelta < 0 {
		absDelta = -absDelta
	}
	surpriseFactor := float32(1.0) + absDelta

	neuroplasticity := genomeNeuroplasticity * (1 + neuroplasticityMod) * surpriseFactor
	if neuroplasticity < 0 {
		neuroplasticity = 0
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
				if neuron := c.Nnet.HiddenNeurons[key]; neuron != nil && neuron.Driven {
					sum := neuronAccumulators[key] * neuron.Sensitivity

					output := float32(math.Tanh(float64(sum)))
					neuron.Output = output
					absOutput := float32(math.Abs(float64(output)))
					neuron.AverageOutput = (neuron.AverageOutput * 0.99) + (absOutput * 0.01)
					if absOutput < 0 {
						absOutput = -absOutput
					}
					c.Energy -= absOutput * energyCostOfFiring
				}
			}
			neuronOutputsEvaluated = true
		}

		var inputVal float32
		if gene.SourceType == SENSOR {
			inputVal = sensorCache[gene.SourceID]
		} else {
			// Cleaned up hidden neuron access
			if neuron := c.Nnet.HiddenNeurons[gene.SourceID]; neuron != nil {
				inputVal = neuron.Output
			}
		}

		currentWeight := c.Nnet.Weights[i]
		isAction := gene.SinkType == ACTION
		isNeuron := gene.SinkType == NEURON

		if isAction {
			actionLevels[gene.SinkID] += inputVal * currentWeight
		} else {
			neuronAccumulators[gene.SinkID] += inputVal * currentWeight
		}

		if isNeuron || isAction {
			var sinkOutput float32
			if isNeuron {
				if neuron := c.Nnet.HiddenNeurons[gene.SinkID]; neuron != nil {
					sinkOutput = neuron.Output
				}
			} else {
				sinkOutput = actionLevels[gene.SinkID]
			}

			correlation := inputVal * sinkOutput
			energyThreshold := c.MaxEnergy(params) * 0.6

			absDopamine := c.Dopamine
			if absDopamine < 0 {
				absDopamine = -absDopamine
			}
			if c.Energy > energyThreshold {
				learningSignal := correlation * softsign(dopamineDelta)
				absSignal := learningSignal
				if absSignal < 0 {
					absSignal = -absSignal
				}
				if absSignal > learningThreshold {
					c.Nnet.Weights[i] += neuroplasticity * learningSignal
					c.Energy -= energyCostOfLearning

					if c.Nnet.Weights[i] > 4.0 {
						c.Nnet.Weights[i] = 4.0
					} else if c.Nnet.Weights[i] < -4.0 {
						c.Nnet.Weights[i] = -4.0
					}
				}
			}
		}
	}

	var weightSumsNeurons [256]float32
	var weightSumsActions [ACTION_COUNT]float32

	for i, gene := range c.Nnet.Edges {
		absWeight := float32(math.Abs(float64(c.Nnet.Weights[i])))
		if gene.SinkType == NEURON {
			weightSumsNeurons[gene.SinkID] += absWeight
		} else {
			weightSumsActions[gene.SinkID] += absWeight
		}
	}

	const neuronBudget = 8.0
	const actionBudget = 12.0 // Actions get a bit more "room" to fire strongly

	for i, gene := range c.Nnet.Edges {
		if gene.SinkType == NEURON {
			total := weightSumsNeurons[gene.SinkID]
			if total > neuronBudget {
				c.Nnet.Weights[i] *= (neuronBudget / total)
			}
		} else {
			total := weightSumsActions[gene.SinkID]
			if total > actionBudget {
				c.Nnet.Weights[i] *= (actionBudget / total)
			}
		}
	}

	const targetActivity = 0.4 // We want the neuron firing at 40% intensity on average
	const adjustmentSpeed = 0.001
	for _, key := range c.Nnet.HiddenNeuronIDs {
		neuron := c.Nnet.HiddenNeurons[key]
		if neuron == nil {
			continue
		}

		// Error = Target - Actual
		// If Actual > Target, error is negative, Sensitivity decreases.
		// If Actual < Target, error is positive, Sensitivity increases.
		neuron.Sensitivity += (targetActivity - neuron.AverageOutput) * adjustmentSpeed

		// Clamp sensitivity so it doesn't go to zero or infinity
		if neuron.Sensitivity < 0.1 {
			neuron.Sensitivity = 0.1
		}
		if neuron.Sensitivity > 5.0 {
			neuron.Sensitivity = 5.0
		}
	}
	c.LastDopamine = c.Dopamine
	return actionLevels
}

func softsign(x float32) float32 {
	if x >= 0 {
		return x / (1 + x)
	}
	return x / (1 - x)
}
