# BioGo

Inspired by https://github.com/davidrmiller/biosim4/tree/main/src

This project is an emergent ecosystem simulation driven by genetic evolution and neural networks. 

'Creatures' compete for survival in a real-time environment closed-energy system, where "plant" food is spawned to replenish energy
lost through metabolic proceses. The world (as of writing) contains a temperature gradient, which affects the metabolic rate and speed of creatures. 

Creatures are defined by their Genome, a series of bytes that determine their characteristics, such as mass, how far they can see, their reproductive type, how much mass they donate to their offspring when reproducing. The genome also defines a brain - a series of sensors to read from the world, and a series of actions to affect it. These are connected into a neural network, the size and complexity of which is again defined genetically. As creatures reproduce, their offspring have a chance to undergo genetic mutation, which will slightly alter the genetic makeup, shifting traits in one direction or another, or even redefining the size and wiring of the brain itself, leading to emergent behaviour.

A genome may look something like this:
```
00101011|01001010|00000111|11110110|01010000|1|00000010|0|00000110|1|10110110|10000111|1|10110110|0|01110001|10101110
```
where each section of bits represents a single characteristic:
```
OscPeriod|Mass|VisionRadius|Responsiveness|MutationRate|ReproductionType|CognitiveBreadth|SynapticDensity|...NeuralGenes
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

Once alive, creatures fine-tune their neural networks via Hebbian learning. A creature may develop an ability to reward itself via dopamine, enabling pathways to be strengthened when desired behaviour is enacted. 

![Screenshot of the simulation.](./images/screen-capture.PNG)

## Performance

This project is equally an exercise in performance as much as it is in exploring artificial intelligence. 

To handle nearly 20,000 creatures at 30fps, each performing feed-forward execution of a neural network, a number of optimisations have been implemented.


### Install
`
cd biogo/
go run .
`
Parameters can be adjusted in biogo/v2/simulation/parameters.go

#### Requirements
Go 1.24.0

Dependencies:
- github.com/hajimehoshi/ebiten/v2 v2.9.9
- golang.org/x/image v0.31.0
