package neuralnet


type NeuralNet struct {
	Edges []*Gene
	HiddenNeurons map[byte]*Neuron
}

