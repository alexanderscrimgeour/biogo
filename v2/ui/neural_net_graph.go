package ui

import (
	"biogo/v2/simulation"
	"fmt"
	"image/color"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	nnPanelW    = float32(360)
	nnColSensor = float32(140)
	nnColAction = float32(220)
	nnInnerPadX = float32(18)
	nnNodeSpY   = float32(15)
	nnTitleH    = float32(20)
	nnPadding   = float32(6)
	nnNodeR     = float32(4)
	nnBarMaxW   = float32(18)
	nnBarH      = float32(3)
	nnFooterH   = float32(16)
)

// NeuralNetGraph is a Component that renders the creature's neural network.
type NeuralNetGraph struct {
	Font *textv2.GoXFace
	Data simulation.CreatureDetailView
}

// computeHeight returns the panel height given the current neural net data.
func (ng *NeuralNetGraph) computeHeight() float32 {
	nn := ng.Data.NeuralNet
	if len(nn.Edges) == 0 && len(nn.HiddenNeuronIDs) == 0 {
		return 0
	}
	sensorSet := map[byte]bool{}
	actionSet := map[byte]bool{}
	for _, e := range nn.Edges {
		if e.SourceType == simulation.SENSOR {
			sensorSet[e.SourceID] = true
		}
		if e.SinkType == simulation.ACTION {
			actionSet[e.SinkID] = true
		}
	}
	neurons := nn.HiddenNeuronIDs
	const maxLayerDepth = 8
	neuronDepth := map[byte]int{}
	for _, id := range neurons {
		neuronDepth[id] = 0
	}
	for _, e := range nn.Edges {
		if e.SourceType == simulation.SENSOR && e.SinkType == simulation.NEURON {
			if neuronDepth[e.SinkID] < 1 {
				neuronDepth[e.SinkID] = 1
			}
		}
	}
	for iter := 0; iter < maxLayerDepth; iter++ {
		changed := false
		for _, e := range nn.Edges {
			if e.SourceType == simulation.NEURON && e.SinkType == simulation.NEURON && e.SourceID != e.SinkID {
				newD := neuronDepth[e.SourceID] + 1
				if newD > maxLayerDepth {
					newD = maxLayerDepth
				}
				if newD > neuronDepth[e.SinkID] {
					neuronDepth[e.SinkID] = newD
					changed = true
				}
			}
		}
		if !changed {
			break
		}
	}
	byDepth := map[int][]byte{}
	for _, id := range neurons {
		dep := neuronDepth[id]
		byDepth[dep] = append(byDepth[dep], id)
	}
	maxCount := len(sensorSet)
	if len(actionSet) > maxCount {
		maxCount = len(actionSet)
	}
	for _, ns := range byDepth {
		if len(ns) > maxCount {
			maxCount = len(ns)
		}
	}
	h := (nnTitleH + nnPadding) + float32(maxCount)*nnNodeSpY + nnPadding + nnFooterH
	if h < 40 {
		h = 40
	}
	return h
}

// Size returns the dimensions of the neural net panel.
func (ng *NeuralNetGraph) Size() (float32, float32) {
	return nnPanelW, ng.computeHeight()
}

// Draw renders the neural net graph at (x, y).
func (ng *NeuralNetGraph) Draw(screen *ebiten.Image, x, y float32) (float32, float32) {
	nn := ng.Data.NeuralNet
	if len(nn.Edges) == 0 && len(nn.HiddenNeuronIDs) == 0 {
		return 0, 0
	}

	sensorSet := map[byte]bool{}
	actionSet := map[byte]bool{}
	for _, e := range nn.Edges {
		if e.SourceType == simulation.SENSOR {
			sensorSet[e.SourceID] = true
		}
		if e.SinkType == simulation.ACTION {
			actionSet[e.SinkID] = true
		}
	}
	sensors := sortedKeys(sensorSet)
	actions := sortedKeys(actionSet)
	neurons := nn.HiddenNeuronIDs

	const maxLayerDepth = 8
	neuronDepth := map[byte]int{}
	for _, id := range neurons {
		neuronDepth[id] = 0
	}
	for _, e := range nn.Edges {
		if e.SourceType == simulation.SENSOR && e.SinkType == simulation.NEURON {
			if neuronDepth[e.SinkID] < 1 {
				neuronDepth[e.SinkID] = 1
			}
		}
	}
	for iter := 0; iter < maxLayerDepth; iter++ {
		changed := false
		for _, e := range nn.Edges {
			if e.SourceType == simulation.NEURON && e.SinkType == simulation.NEURON && e.SourceID != e.SinkID {
				newD := neuronDepth[e.SourceID] + 1
				if newD > maxLayerDepth {
					newD = maxLayerDepth
				}
				if newD > neuronDepth[e.SinkID] {
					neuronDepth[e.SinkID] = newD
					changed = true
				}
			}
		}
		if !changed {
			break
		}
	}

	byDepth := map[int][]byte{}
	numDepths := 0
	for _, id := range neurons {
		dep := neuronDepth[id]
		byDepth[dep] = append(byDepth[dep], id)
		if dep+1 > numDepths {
			numDepths = dep + 1
		}
	}
	for dep := range byDepth {
		sort.Slice(byDepth[dep], func(i, j int) bool { return byDepth[dep][i] < byDepth[dep][j] })
	}

	neuronColX := map[byte]float32{}
	innerLeft := nnColSensor + nnInnerPadX
	innerRight := nnColAction - nnInnerPadX
	innerSpan := innerRight - innerLeft
	for id, dep := range neuronDepth {
		frac := float32(dep+1) / float32(numDepths+1)
		neuronColX[id] = innerLeft + innerSpan*frac
	}

	maxCount := len(sensors)
	if len(actions) > maxCount {
		maxCount = len(actions)
	}
	for _, ns := range byDepth {
		if len(ns) > maxCount {
			maxCount = len(ns)
		}
	}
	panelH := (nnTitleH + nnPadding) + float32(maxCount)*nnNodeSpY + nnPadding + nnFooterH
	if panelH < 40 {
		panelH = 40
	}

	vector.FillRect(screen, x, y, nnPanelW, panelH, color.RGBA{8, 10, 22, 215}, false)
	vector.StrokeRect(screen, x, y, nnPanelW, panelH, 1, color.RGBA{90, 90, 150, 255}, false)
	if ng.Font != nil {
		drawText(screen, "NEURAL NETWORK", ng.Font, int(x+nnPadding), int(y)+2, color.RGBA{120, 120, 180, 255})
	}

	contentTop := y + nnTitleH + nnPadding
	contentH := panelH - (nnTitleH + nnPadding*2) - nnFooterH

	nodeY := func(count, i int) float32 {
		if count == 0 {
			return contentTop + contentH/2
		}
		step := contentH / float32(count)
		return contentTop + step*float32(i) + step/2
	}

	sensorY := map[byte]float32{}
	for i, s := range sensors {
		sensorY[s] = nodeY(len(sensors), i)
	}
	neuronY := map[byte]float32{}
	for _, ns := range byDepth {
		for i, id := range ns {
			neuronY[id] = nodeY(len(ns), i)
		}
	}
	actionY := map[byte]float32{}
	for i, a := range actions {
		actionY[a] = nodeY(len(actions), i)
	}

	for _, e := range nn.Edges {
		var srcX, srcY, dstX, dstY float32
		switch e.SourceType {
		case simulation.SENSOR:
			srcX = x + nnColSensor
			srcY = sensorY[e.SourceID]
		case simulation.NEURON:
			srcX = x + neuronColX[e.SourceID]
			srcY = neuronY[e.SourceID]
		}
		switch e.SinkType {
		case simulation.NEURON:
			dstX = x + neuronColX[e.SinkID]
			dstY = neuronY[e.SinkID]
		case simulation.ACTION:
			dstX = x + nnColAction
			dstY = actionY[e.SinkID]
		}
		if e.SourceType == simulation.NEURON && e.SinkType == simulation.NEURON && e.SourceID == e.SinkID {
			lx := x + neuronColX[e.SourceID]
			ly := neuronY[e.SourceID] - nnNodeR*2
			var path vector.Path
			path.Arc(lx, ly, nnNodeR*1.5, 0, 2*math.Pi, vector.Clockwise)
			lc := nnEdgeColor(e.Weight)
			var cs ebiten.ColorScale
			cs.ScaleWithColor(lc)
			vector.StrokePath(screen, &path, &vector.StrokeOptions{Width: 1}, &vector.DrawPathOptions{ColorScale: cs})
			continue
		}
		vector.StrokeLine(screen, srcX, srcY, dstX, dstY, 1, nnEdgeColor(e.Weight), false)
	}

	for _, s := range sensors {
		nx := x + nnColSensor
		ny := sensorY[s]
		vector.FillCircle(screen, nx, ny, nnNodeR, color.RGBA{80, 150, 220, 255}, false)

		lbl := nnSensorName(s)

		if ng.Font != nil {
			metrics := ng.Font.Metrics()
			textHeight := float32(metrics.HAscent + metrics.HDescent)

			tw, _ := textv2.Measure(lbl, ng.Font, 0)
			lblW := float32(tw)

			labelX := (nx - 10) - lblW
			if labelX < x+4 {
				labelX = x + 4
			}

			drawText(screen, lbl, ng.Font, int(labelX), int(ny-textHeight/2), color.RGBA{160, 180, 220, 255})
		}

		if val, ok := nn.SensorValues[s]; ok {
			barX := x + nnPadding
			barY := ny - nnBarH/2
			centerX := barX + nnBarMaxW/2
			vector.FillRect(screen, barX, barY, nnBarMaxW, nnBarH, color.RGBA{20, 40, 70, 180}, false)
			vector.StrokeLine(screen, centerX, barY, centerX, barY+nnBarH, 1, color.RGBA{255, 255, 255, 50}, false)

			cVal := val
			if cVal > 1 {
				cVal = 1
			} else if cVal < -1 {
				cVal = -1
			}

			halfWidth := (nnBarMaxW / 2) * cVal
			var fillColor color.RGBA
			var fillX float32
			if halfWidth >= 0 {
				fillX = centerX
				fillColor = color.RGBA{80, 160, 240, 220}
			} else {
				fillX = centerX + halfWidth
				halfWidth = -halfWidth
				fillColor = color.RGBA{240, 80, 80, 220}
			}
			if halfWidth > 0 {
				vector.FillRect(screen, fillX, barY, halfWidth, nnBarH, fillColor, false)
			}
		}
	}

	for _, id := range neurons {
		nx := x + neuronColX[id]
		ny := neuronY[id]
		vector.FillCircle(screen, nx, ny, nnNodeR, color.RGBA{200, 180, 80, 255}, false)
	}
	for _, a := range actions {
		ax := x + nnColAction
		ay := actionY[a]
		vector.FillCircle(screen, ax, ay, nnNodeR, color.RGBA{220, 100, 80, 255}, false)
		if ng.Font != nil {
			metrics := ng.Font.Metrics()
			textHeight := float32(metrics.HAscent + metrics.HDescent)
			drawText(screen, nnActionName(a), ng.Font, int(ax)+10, int(ay-textHeight/2), color.RGBA{220, 160, 150, 255})
		}
		if val, ok := nn.ActionValues[a]; ok {
			barX := x + nnPanelW - nnPadding - nnBarMaxW
			barY := ay - nnBarH/2
			norm := float32(float64(val) / (1.0 + math.Abs(float64(val))))
			vector.FillRect(screen, barX, barY, nnBarMaxW, nnBarH, color.RGBA{60, 25, 15, 180}, false)
			centerX := barX + nnBarMaxW/2
			vector.StrokeLine(screen, centerX, barY, centerX, barY+nnBarH, 1, color.RGBA{255, 255, 255, 50}, false)
			if norm != 0 {
				halfMaxW := nnBarMaxW / 2
				fillW := halfMaxW * norm
				fc := color.RGBA{240, 130, 80, 220}
				if norm < 0 {
					fc = color.RGBA{80, 130, 240, 220}
				}
				drawX := centerX
				if norm < 0 {
					drawX = centerX + fillW
				}
				if fillW < 0 {
					fillW = -fillW
				}
				vector.FillRect(screen, drawX, barY, fillW, nnBarH, fc, false)
			}
		}
	}

	const baseNeuroplasticity = 0.01
	footerY := int(y + panelH - nnFooterH)
	if ng.Font != nil {
		drawText(screen, fmt.Sprintf("Learning Rate: %.4f", baseNeuroplasticity*ng.Data.Dopamine), ng.Font,
			int(x+nnPadding), footerY, color.RGBA{120, 120, 180, 220})
	}

	return nnPanelW, panelH
}

func sortedKeys(m map[byte]bool) []byte {
	out := make([]byte, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func nnEdgeColor(w float32) color.RGBA {
	absW := float32(math.Abs(float64(w)))
	intensity := absW / 4.0
	if intensity > 1.0 {
		intensity = 1.0
	}
	v := uint8(intensity * 255)
	alpha := uint8(30 + intensity*190)
	if w >= 0 {
		return color.RGBA{R: 10, G: v, B: 10, A: alpha}
	}
	return color.RGBA{R: v, G: 10, B: 10, A: alpha}
}

func nnSensorName(id byte) string {
	names := [...]string{
		"Bias", "Energy", "NearFoodAngle",
		"NearFoodDist", "Age", "LocX", "LocY",
		"Heading", "Velocity", "Osc1", "BlockedFwd",
		"WallProx", "PlantFwd", "MeatFwd", "PreyFwd",
		"NearPreyAngle", "ThreatFwd", "NearThreatAngle",
		"Mass %", "Juvenile", "Satiation", "StomachRate",
		"LocalCOM", "LocalDensity", "LocalHeading",
		"PopFwd", "PopDensityFwd", "Touching", "Temp",
		"TempDelta", "EnergyDelta", "Random", "KinshipLocal",
		"KinshipNearDist", "KinshipNear", "LocalFoodPerCapita",
	}
	if int(id) < len(names) {
		return names[id]
	}
	return fmt.Sprintf("S%d", id)
}

func nnActionName(id byte) string {
	names := [...]string{
		"Accelerate", "Rotate", "SetOsc",
		"Rest", "Attack", "Reproduce",
		"Feed", "SetResp",
		"SetLearn", "Reward", "Punish",
	}
	if int(id) < len(names) {
		return names[id]
	}
	return fmt.Sprintf("A%d", id)
}
