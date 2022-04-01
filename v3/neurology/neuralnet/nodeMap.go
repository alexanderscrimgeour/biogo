package neuralnet


type Node struct {
	NewID byte
	OutputCount byte
	InputCount byte
	SelfLoopCount byte
}

type NodeMap map[byte]*Node

func CreateInitialNeuronOutput() float32 { return 0.5 }

func CreateNeuralNetworkFromGenome()