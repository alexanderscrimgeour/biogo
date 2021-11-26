## BioGo

This is a simple genetic simulator written in Go.

Heavily inspired by https://github.com/davidrmiller/biosim4/tree/main/src


![Generation 508](./images/generation-508.gif)

#### About

This project simulates natural selection and genetic inheritence. 

To start, we represent behaviour via a genome, which is a collection of bytes that determine general characteristics, as well as a slice of genes responsible for generating a neural network.

E.g. A genome may look something like this:\

`
00101011|01001010|00000111|11110110|01010000|1|00000010|0|00000110|1|10110110|10000111|1|10110110|0|01110001|10101110
`
where each section of bits represents a single characteristic:\
`OscPeriod|MaxEnergy|SightDistance|Responsiveness|MutationRate|ReproductionType|NeuronCount|BrainLength|...NeuralGenes
`
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

At the start of the simulation a number of creatures are created with random genomes. The creatures must solve a challenge, e.g. reach the left 10% of the map, before the "generation" is over. Those that achieve this go on to populate the next generation.

A creature's offspring has a chance of genetic mutation when inheriting the parent's genome.

In the example presented above, by the 500th generation 99.6% of creatures managed to survive the challenge and reach the left hand side.

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
- Continuous environment rather than "Generations"
- Redo the hastily made test UI
- Output data to file for analysis