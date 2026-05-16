package simulation

import (
	"biogo/v2/world"
)

const energyCostOfFiring = 0.0001
const decayRate = 0.0005
const energyCostOfLearning = 0.005

func (c *Creature) FeedForward(w *world.World, p *Population, step int, params *Parameters) {
	neuroplasticityMod := softsign(c.Nnet.LastActionValues[SET_LEARNING_RATE])

	c.Nnet.LastActionValues = [ACTION_COUNT]float32{}
	actionLevels := &c.Nnet.LastActionValues

	var neuronAccumulators [256]float32

	// update the SensorContext once to eliminate duplicate spatial queries
	c.UpdateSensorContext(w, p, params)
	ctx := &c.Sensors

	var sensorCache [SENSOR_COUNT]float32
	c.Nnet.LastSensorValues = [SENSOR_COUNT]float32{}
	for sid := byte(0); sid < SENSOR_COUNT; sid++ {
		if c.Nnet.ActiveSensors[sid] {
			val := c.GetSensor(sid, w, p, ctx, step, params)
			sensorCache[sid] = val
			c.Nnet.LastSensorValues[sid] = val
		}
	}

	genomeNeuroplasticity := params.MinNeuroplasticity + float32(c.Genome.Neuroplasticity)/255.0*(params.MaxNeuroplasticity-params.MinNeuroplasticity)
	learningThreshold := params.MinLearningThreshold + float32(c.Genome.LearningThreshold)/255.0*(params.MaxLearningThreshold-params.MinLearningThreshold)

	dopamineDelta := c.Dopamine - c.LastDopamine
	surpriseFactor := float32(1.0) + absf32(dopamineDelta)
	dopamineSoftSign := softsign(dopamineDelta)

	neuroplasticity := genomeNeuroplasticity * (1 + neuroplasticityMod) * surpriseFactor
	if neuroplasticity < 0 {
		neuroplasticity = 0
	}

	energyThreshold := c.MaxEnergy(params) * 0.6

	// Phase 1: neuron-sink edges. Neurons haven't fired yet so neuron.Output is
	// the previous step's value — Hebbian uses that as sinkOutput intentionally.
	for i := 0; i < c.Nnet.NeuronEdgeCount; i++ {
		gene := c.Nnet.Edges[i]

		instinctWeight := gene.WeightAsFloat32()
		if c.Nnet.Weights[i] > instinctWeight {
			c.Nnet.Weights[i] -= decayRate
		} else if c.Nnet.Weights[i] < instinctWeight {
			c.Nnet.Weights[i] += decayRate
		}

		var inputVal float32
		if gene.SourceType == SENSOR {
			inputVal = sensorCache[gene.SourceID]
		} else {
			inputVal = c.Nnet.HiddenNeurons[gene.SourceID].Output
		}

		neuronAccumulators[gene.SinkID] += inputVal * c.Nnet.Weights[i]

		if c.Energy > energyThreshold {
			sinkOutput := c.Nnet.HiddenNeurons[gene.SinkID].Output
			learningSignal := inputVal * sinkOutput * dopamineSoftSign
			if absf32(learningSignal) > learningThreshold {
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

	// Evaluate hidden neurons now that all neuron-sink inputs are accumulated.
	for key := range c.Nnet.HiddenNeurons {
		neuron := &c.Nnet.HiddenNeurons[key]
		if neuron.Driven {
			sum := neuronAccumulators[key] * neuron.Sensitivity
			output := tanhf(sum)
			neuron.Output = output
			absOutput := absf32(output)
			neuron.AverageOutput = (neuron.AverageOutput * 0.99) + (absOutput * 0.01)
			c.Energy -= absOutput * energyCostOfFiring
		}
	}

	// Phase 2: action-sink edges. Neuron outputs are current-step values.
	for i := c.Nnet.NeuronEdgeCount; i < len(c.Nnet.Edges); i++ {
		gene := c.Nnet.Edges[i]

		instinctWeight := gene.WeightAsFloat32()
		if c.Nnet.Weights[i] > instinctWeight {
			c.Nnet.Weights[i] -= decayRate
		} else if c.Nnet.Weights[i] < instinctWeight {
			c.Nnet.Weights[i] += decayRate
		}

		var inputVal float32
		if gene.SourceType == SENSOR {
			inputVal = sensorCache[gene.SourceID]
		} else {
			inputVal = c.Nnet.HiddenNeurons[gene.SourceID].Output
		}

		actionLevels[gene.SinkID] += inputVal * c.Nnet.Weights[i]

		if c.Energy > energyThreshold {
			sinkOutput := actionLevels[gene.SinkID]
			learningSignal := inputVal * sinkOutput * dopamineSoftSign
			if absf32(learningSignal) > learningThreshold {
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

	if step%10 == 0 {
		c.normalizeWeights()
	}

	const targetActivity = 0.4 // We want the neuron firing at 40% intensity on average
	const adjustmentSpeed = 0.001
	for key := range c.Nnet.HiddenNeurons {
		neuron := &c.Nnet.HiddenNeurons[key]

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
}

// normalizeWeights rescales edge weights so no sink's total |weight| exceeds
// its budget. Called every 10 steps rather than every step.
func (c *Creature) normalizeWeights() {
	var weightSumsNeurons [256]float32
	var weightSumsActions [ACTION_COUNT]float32

	for i, gene := range c.Nnet.Edges {
		aw := absf32(c.Nnet.Weights[i])
		if gene.SinkType == NEURON {
			weightSumsNeurons[gene.SinkID] += aw
		} else {
			weightSumsActions[gene.SinkID] += aw
		}
	}

	const neuronBudget = 8.0
	const actionBudget = 12.0
	for i, gene := range c.Nnet.Edges {
		if gene.SinkType == NEURON {
			if total := weightSumsNeurons[gene.SinkID]; total > neuronBudget {
				c.Nnet.Weights[i] *= neuronBudget / total
			}
		} else {
			if total := weightSumsActions[gene.SinkID]; total > actionBudget {
				c.Nnet.Weights[i] *= actionBudget / total
			}
		}
	}
}

