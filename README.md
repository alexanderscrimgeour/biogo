## BioGo

This is a simple genetic simulator written in Go.

Heavily inspired by https://github.com/davidrmiller/biosim4/tree/main/src


#### About

The simulation creates Creatures with random genomes, and gives them a challenge to solve.

Currently the challenges are very simple, e.g. Creatures on the left 10% of the map at the end of one generation survive.

Those that survive go on to populate the next generation. Eventually, the creatures learn to overcome the challenge and survive:

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