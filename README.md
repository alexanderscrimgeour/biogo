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

At the start of the simulation a number of creatures are created with random genomes. The creatures must solve a challenge, e.g. reach the left 10% of the map, before the "generation" is over. 
At first, the creatures are terrible at solving the problem:
![Generation 1](./images/generation-1.gif)

However, those that succeed go on to populate the next generation. Eventually, those that are most able to move left are able to dominate the genetic makeup of the population. However, some of these creatures are still blocked by the wall, and as such the survival rate stabilises at around 70%.
![Generation 1](./images/generation-51.gif)

The final piece of the puzzle is that a creature's offspring has a chance of genetic mutation when inheriting the parent's genome. This means that eventually, a more optimal strategy will emerge, and this new genome will take over. In this test, a mutation occured that allowed the children to first travel down to below the wall before heading left.
By the 500th generation 99.6% of creatures managed to survive the challenge and reach the left hand side:
![Generation 508](./images/generation-508.gif)

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