package ui

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	histGraphW   = float32(220)
	histGraphH   = float32(120)
	histGraphPad = float32(4)
)

// HistoryGraph renders population/food/energy time-series as a line chart.
// It reads game state via closures set at construction time.
type HistoryGraph struct {
	getCount  func() int
	getHead   func() int
	getSample func(i int) histSample
	maxEnergy float64
}

// Draw renders the line graph panel at (x, y).
func (hg *HistoryGraph) Draw(screen *ebiten.Image, x, y float32) (float32, float32) {
	count := hg.getCount()
	panelFill := color.RGBA{8, 10, 22, 160}
	panelStroke := color.RGBA{50, 60, 90, 180}
	vector.FillRect(screen, x, y, histGraphW, histGraphH, panelFill, false)
	vector.StrokeRect(screen, x, y, histGraphW, histGraphH, 1, panelStroke, false)

	if count < 2 {
		return histGraphW, histGraphH
	}

	popColor := color.RGBA{100, 180, 255, 255}
	foliageColor := color.RGBA{80, 210, 100, 255}
	fungiColor := color.RGBA{150, 50, 190, 255}
	meatColor := color.RGBA{210, 90, 90, 255}
	energyColor := color.RGBA{255, 230, 50, 255}

	gx := x + histGraphPad
	gy := y + histGraphPad
	gw := histGraphW - histGraphPad*2
	gh := histGraphH - histGraphPad*2

	head := hg.getHead()
	popMax := 1
	var foodEnergyMax float64 = 1
	for i := 0; i < count; i++ {
		idx := ((head-1-i)%historyLen + historyLen) % historyLen
		s := hg.getSample(idx)
		if s.pop > popMax {
			popMax = s.pop
		}
		if s.foliageEnergy > foodEnergyMax {
			foodEnergyMax = s.foliageEnergy
		}
		if s.fungiEnergy > foodEnergyMax {
			foodEnergyMax = s.fungiEnergy
		}
		if s.meatEnergy > foodEnergyMax {
			foodEnergyMax = s.meatEnergy
		}
		if s.totalEnergy > hg.maxEnergy {
			hg.maxEnergy = s.totalEnergy
		}
	}

	steps := int(gw)
	if steps > count {
		steps = count
	}

	foodEnergyScale := int(foodEnergyMax * 2)
	hg.drawLine(screen, gx, gy, gw, gh, steps, head, count, foodEnergyScale, foliageColor,
		func(s histSample) int { return int(s.foliageEnergy) })
	hg.drawLine(screen, gx, gy, gw, gh, steps, head, count, foodEnergyScale, fungiColor,
		func(s histSample) int { return int(s.fungiEnergy) })
	hg.drawLine(screen, gx, gy, gw, gh, steps, head, count, foodEnergyScale, meatColor,
		func(s histSample) int { return int(s.meatEnergy) })
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

// Size returns the fixed dimensions of the history line graph panel.
func (hg *HistoryGraph) Size() (float32, float32) {
	return histGraphW, histGraphH
}
