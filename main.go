package main

import (
	"biogo/v2/simulation"
	"biogo/v2/ui"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	go func() {
		// Using "0.0.0.0" instead of "localhost" can sometimes bypass
		// IPv6 vs IPv4 issues (the [::1] in your error)
		log.Println(http.ListenAndServe("0.0.0.0:6060", nil))
	}()
	rand.Seed(time.Now().UnixNano())
	params := simulation.DefaultParams()
	sim := simulation.New(params)

	game := ui.NewGame(sim)

	ebiten.SetWindowSize(1280, 800)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("Genetic Simulation")

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
