package main

import (
	"biogo/v2/simulation"
	"biogo/v2/ui"
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"

	"github.com/hajimehoshi/ebiten/v2"
)

// pprofAddr is where the profiling server binds when enabled. It stays on
// localhost so profiles are never exposed on other network interfaces.
const pprofAddr = "localhost:6060"

func main() {
	enablePprof := flag.Bool("pprof", false, "serve net/http/pprof on "+pprofAddr)
	flag.Parse()

	if *enablePprof {
		go func() {
			log.Printf("pprof listening on http://%s/debug/pprof/", pprofAddr)
			log.Println(http.ListenAndServe(pprofAddr, nil))
		}()
	}

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
