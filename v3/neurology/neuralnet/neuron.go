package neuralnet

const (
	NEURON = 0 // Source & Sink
	SENSOR = 1 // Source
	ACTION = 1 // Sink
)

// Neurons represent hidden neurons in the net
type Neuron struct {
	Output float32
	Driven bool
}
