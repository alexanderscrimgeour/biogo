## BioGo

This is a simple genetic simulator written in Go.

Heavily inspired by https://github.com/davidrmiller/biosim4/tree/main/src

#### About

This project simulates natural selection and genetic inheritence. 

To start, we represent behaviour via a genome, which is a collection of bytes that determine general characteristics, as well as a slice of genes responsible for generating a neural network.

A genome may look something like this:\

```
00101011|01001010|00000111|11110110|01010000|1|00000010|0|00000110|1|10110110|10000111|1|10110110|0|01110001|10101110
```
where each section of bits represents a single characteristic:\
```
OscPeriod|MaxEnergy|SightDistance|Responsiveness|MutationRate|ReproductionType|NeuronCount|BrainLength|...NeuralGenes
```
with neural genes representing a neural pathway:
```
SourceType|SourceID|SinkType|SinkID  |Weight
----------|--------|--------|--------|--------
         0|00000110|       1|10110110|10000111
         1|10110110|       0|01110001|10101110
```

In the simple case above, we create a simple neural network with three neurons:
```
    |--------|    weight   |--------|   weight    |--------|
    | Sensor |---10000111->| Neuron |--10101110-->| Action |
    |--------|             |--------|             |--------|
```

By encoding the neural network in the genome itself, we are able to mutate the neural network through evolution - including changing the size of the neural network or rewiring connections. 

The simulation runs by generating a continuous stream of food sources that creatures can feed to replenish energy levels. If a creature exhausts it's energy, it dies. Creatures may also eat each other as energy sources.

When creatures reach a certain limit of energy, it may reproduce. Reproduction is currently asexual, creating a cloned child with a chance of mutation (the rate of which is genetic). This ensures that creatures evolve to environmental pressures. 

If the population dips below a minimum value, the simulation spawns new mutant variants of the best performing creature in order to give evolution a helping hand. 

Simply debug tools exist at the moment that should enable saving the 5 best creatures (done via greedy sampling of genetically similar clusters).

### Install
`
cd biogo/
go run .
`
Parameters can be adjusted in biogo/v2/simulation/parameters.go
#### Requirements
Go 1.15

Dependencies:
- github.com/hajimehoshi/ebiten
- github.com/hajimehoshi/ebiten/v2
- golang.org/x/exp
- golang.org/x/image

#### TODO
- Concurrent execution of neural FFward steps to improve performance
- Consider ECS redesign if performance becomes an issue
- Visualise neural networks, creature splits, etc. 
- Redo the hastily made test UI