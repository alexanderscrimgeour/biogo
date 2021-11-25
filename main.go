package main

import (
	"fmt"
	"gopop/v2/grid"
	"gopop/v2/simulation"
	"math/rand"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 5000; i++ {
		g := simulation.MakeRandomGenome()
		// g2 := simulation.AsexualReproduction(g)
		// similarity := simulation.GenomeSimilarity(*g, *g2)
		// fmt.Printf("Parent/Child Similarity: %f\n", similarity)
		creature := simulation.NewCreature(i, grid.Coord{X: 0, Y: 0}, g)
		fmt.Println(creature.String())
	}
}
