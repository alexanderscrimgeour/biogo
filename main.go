package main

import (
	"fmt"
	"gopop/v2/simulation"
	"math/rand"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	start := time.Now()
	g := simulation.MakeRandomGenome()
	updating := g
	for i := 0; i < 100000; i++ {
		// go func() {
		newG := simulation.AsexualReproduction(updating)
		similarity := simulation.GenomeSimilarity(*g, *newG)
		fmt.Printf("Parent/Child Similarity: %f\n", similarity)
		// simulation.NewCreature(i, grid.Coord{X: 0, Y: 0}, g)
		// fmt.Println(creature.String())
		// }()
		updating = newG.Copy()
	}
	end := time.Now()

	fmt.Printf("Completed in: %s", end.Sub(start))
}
