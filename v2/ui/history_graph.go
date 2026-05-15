package ui

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	histGraphW     = float32(220)
	histGraphH     = float32(220)
	histGraphTextH = float32(81)
	histGraphPad   = float32(4)
)

// HistoryGraph is a Component that renders population/food/energy time-series.
// It reads game state via closures set at construction time.
type HistoryGraph struct {
	Font      *textv2.GoXFace
	getCount  func() int
	getHead   func() int
	getSample func(i int) histSample
	sim       interface {
		PopulationCount() int
		FoodCount() int
		TotalEnergy() float64
		TargetEnergy() float64
	}
	maxEnergy float64
}

// Draw renders the history graph panel at (x, y).
func (hg *HistoryGraph) Draw(screen *ebiten.Image, x, y float32) (float32, float32) {
	count := hg.getCount()
	if count < 2 {
		// Draw empty panel placeholder
		vector.FillRect(screen, x, y, histGraphW, histGraphH, color.RGBA{8, 10, 22, 160}, false)
		vector.StrokeRect(screen, x, y, histGraphW, histGraphH, 1, color.RGBA{50, 60, 90, 180}, false)
		return histGraphW, histGraphH
	}

	panelFill := color.RGBA{8, 10, 22, 160}
	panelStroke := color.RGBA{50, 60, 90, 180}
	vector.FillRect(screen, x, y, histGraphW, histGraphH, panelFill, false)
	vector.StrokeRect(screen, x, y, histGraphW, histGraphH, 1, panelStroke, false)

	popColor := color.RGBA{100, 180, 255, 255}
	foodColor := color.RGBA{80, 210, 100, 255}
	energyColor := color.RGBA{255, 230, 50, 255}

	if hg.Font != nil {
		drawText(screen, fmt.Sprintf("Pop: %d", hg.sim.PopulationCount()), hg.Font,
			int(x+histGraphPad), int(y+15), popColor)
		drawText(screen, fmt.Sprintf("Food: %d", hg.sim.FoodCount()), hg.Font,
			int(x+histGraphPad), int(y+31), foodColor)
		drawText(screen, fmt.Sprintf("Energy: %.2f%%", hg.sim.TotalEnergy()/hg.sim.TargetEnergy()*100), hg.Font,
			int(x+histGraphPad), int(y+47), energyColor)
	}

	gx := x + histGraphPad
	gy := y + histGraphTextH
	gw := histGraphW - histGraphPad*2
	gh := histGraphH - histGraphTextH - histGraphPad

	head := hg.getHead()
	foodMax, popMax := 1, 1
	for i := 0; i < count; i++ {
		idx := ((head-1-i)%historyLen + historyLen) % historyLen
		s := hg.getSample(idx)
		if s.pop > popMax {
			popMax = s.pop
		}
		if s.food > foodMax {
			foodMax = s.food
		}
		if s.totalEnergy > hg.maxEnergy {
			hg.maxEnergy = s.totalEnergy
		}
	}

	steps := int(gw)
	if steps > count {
		steps = count
	}

	hg.drawLine(screen, gx, gy, gw, gh, steps, head, count, foodMax*2, foodColor,
		func(s histSample) int { return s.food })
	hg.drawLine(screen, gx, gy, gw, gh, steps, head, count, popMax*2, popColor,
		func(s histSample) int { return s.pop })
	if hg.maxEnergy > 0 {
		hg.drawLine(screen, gx, gy, gw, gh, steps, head, count, int(hg.maxEnergy), energyColor,
			func(s histSample) int { return int(s.totalEnergy) })
	}

	return histGraphW, histGraphH
}

func (hg *HistoryGraph) drawLine(screen *ebiten.Image, gx, gy, gw, gh float32, steps, head, count, maxVal int, clr color.RGBA, get func(histSample) int) {
	if steps < 2 || maxVal == 0 {
		return
	}
	var path vector.Path
	for i := 0; i < steps; i++ {
		frac := float64(i) / float64(steps-1)
		logicalPos := float64(head-count) + frac*float64(count-1)
		sampleIdx := ((int(math.Round(logicalPos)) % historyLen) + historyLen) % historyLen
		val := get(hg.getSample(sampleIdx))
		x := gx + float32(i)/float32(steps-1)*gw
		y := gy + gh*(1-float32(val)/float32(maxVal))
		if y < gy {
			y = gy
		}
		if y > gy+gh {
			y = gy + gh
		}
		if i == 0 {
			path.MoveTo(x, y)
		} else {
			path.LineTo(x, y)
		}
	}
	var cs ebiten.ColorScale
	cs.ScaleWithColor(clr)
	vector.StrokePath(screen, &path, &vector.StrokeOptions{Width: 1.5}, &vector.DrawPathOptions{ColorScale: cs})
}

// Size returns the fixed dimensions of the history graph panel.
func (hg *HistoryGraph) Size() (float32, float32) {
	return histGraphW, histGraphH
}
