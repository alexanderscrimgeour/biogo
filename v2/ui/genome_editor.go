package ui

import (
	"biogo/v2/simulation"
	"biogo/v2/ui/components"
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// ── Panel geometry ────────────────────────────────────────────────────────────

const (
	gePanW  = 900
	gePanH  = 780 // extra height accommodates the filter section below traits
	geTitleH = 28
	geFooterH = 80
	gePad    = 10

	geTraitW       = 240
	geNNSectionX   = gePad + geTraitW + 16         // start of NN section within panel
	geNNW          = gePanW - geNNSectionX - gePad
	geNNSubtitleH  = 22
	geNNControlH   = 72
	geTraitRowH    = 26
	geTraitLabelW  = 98
	geTraitTrackW  = 100
	geTraitTrackX0 = geTraitLabelW + 6

	// Node column offsets within the NN section
	geNNSensorX = 130
	geNNNeuronX = 310
	geNNActionX = 510

	geNodeR    float32 = 5
	geEdgeMidR float32 = 4

	// Filter section (below traits in left column)
	geFilterRowH    = float32(13) // height of each checkbox row in the filter section
	geFilterMaxSens = 14          // max visible sensor rows before scrolling
)

// ── Trait definitions ─────────────────────────────────────────────────────────

type traitDef struct {
	label    string
	get      func(*simulation.Genome) byte
	set      func(*simulation.Genome, byte)
	minB     func(*simulation.Genome, *simulation.Parameters) byte
	maxB     func(*simulation.Genome, *simulation.Parameters) byte
	isToggle bool
}

func makeTraitDefs() []traitDef {
	return []traitDef{
		{
			label: "Osc Period",
			get:   func(g *simulation.Genome) byte { return g.OscPeriod },
			set:   func(g *simulation.Genome, v byte) { g.OscPeriod = v },
			minB:  func(_ *simulation.Genome, _ *simulation.Parameters) byte { return 1 },
			maxB:  func(_ *simulation.Genome, _ *simulation.Parameters) byte { return 255 },
		},
		{
			label: "Sight Dist",
			get:   func(g *simulation.Genome) byte { return g.VisionRadius },
			set:   func(g *simulation.Genome, v byte) { g.VisionRadius = v },
			minB:  func(_ *simulation.Genome, _ *simulation.Parameters) byte { return 0 },
			maxB:  func(_ *simulation.Genome, _ *simulation.Parameters) byte { return 255 },
		},
		{
			label: "Field of View",
			get:   func(g *simulation.Genome) byte { return g.FieldOfView },
			set:   func(g *simulation.Genome, v byte) { g.FieldOfView = v },
			minB:  func(_ *simulation.Genome, _ *simulation.Parameters) byte { return 0 },
			maxB:  func(_ *simulation.Genome, _ *simulation.Parameters) byte { return 255 },
		},
		{
			label: "Responsive",
			get:   func(g *simulation.Genome) byte { return g.Responsiveness },
			set:   func(g *simulation.Genome, v byte) { g.Responsiveness = v },
			minB:  func(_ *simulation.Genome, _ *simulation.Parameters) byte { return 0 },
			maxB:  func(_ *simulation.Genome, _ *simulation.Parameters) byte { return 255 },
		},
		{
			label: "Mutation",
			get:   func(g *simulation.Genome) byte { return g.MutationRate },
			set:   func(g *simulation.Genome, v byte) { g.MutationRate = v },
			minB:  func(_ *simulation.Genome, _ *simulation.Parameters) byte { return 1 },
			maxB:  func(_ *simulation.Genome, _ *simulation.Parameters) byte { return 255 },
		},
		{
			label: "Mass",
			get:   func(g *simulation.Genome) byte { return g.BodyMass },
			set: func(g *simulation.Genome, v byte) {
				g.BodyMass = v
				maxMin := (v - 1) / 2
				if maxMin < 1 {
					maxMin = 1
				}
				if g.SurvivalMass > maxMin {
					g.SurvivalMass = maxMin
				}
			},
			minB: func(_ *simulation.Genome, _ *simulation.Parameters) byte { return 3 },
			maxB: func(_ *simulation.Genome, _ *simulation.Parameters) byte { return math.MaxUint8 },
		},
		{
			label: "Min Mass",
			get:   func(g *simulation.Genome) byte { return g.SurvivalMass },
			set:   func(g *simulation.Genome, v byte) { g.SurvivalMass = v },
			minB:  func(_ *simulation.Genome, _ *simulation.Parameters) byte { return 1 },
			maxB: func(g *simulation.Genome, _ *simulation.Parameters) byte {
				m := (g.BodyMass - 1) / 2
				if m < 1 {
					m = 1
				}
				return m
			},
		},
		{
			label:    "Repro Type",
			get:      func(g *simulation.Genome) byte { return g.ReproductionType },
			set:      func(g *simulation.Genome, v byte) { g.ReproductionType = v },
			minB:     func(_ *simulation.Genome, _ *simulation.Parameters) byte { return 0 },
			maxB:     func(_ *simulation.Genome, _ *simulation.Parameters) byte { return 1 },
			isToggle: true,
		},
		{
			label: "Neuron Count",
			get:   func(g *simulation.Genome) byte { return g.CognitiveBreadth },
			set:   func(g *simulation.Genome, v byte) { g.CognitiveBreadth = v },
			minB:  func(_ *simulation.Genome, p *simulation.Parameters) byte { return p.Neurology.MinSynapticDensity },
			maxB:  func(_ *simulation.Genome, p *simulation.Parameters) byte { return p.Neurology.MaxSynapticDensity },
		},
		{
			label: "Brain Length",
			get:   func(g *simulation.Genome) byte { return g.SynapticDensity },
			set:   func(g *simulation.Genome, v byte) { g.SynapticDensity = v },
			minB:  func(_ *simulation.Genome, p *simulation.Parameters) byte { return p.Neurology.MinSynapticDensity },
			maxB:  func(_ *simulation.Genome, p *simulation.Parameters) byte { return p.Neurology.MaxSynapticDensity },
		},
		{
			label: "Juvenile",
			get:   func(g *simulation.Genome) byte { return g.JuvenilePeriod },
			set:   func(g *simulation.Genome, v byte) { g.JuvenilePeriod = v },
			minB:  func(_ *simulation.Genome, _ *simulation.Parameters) byte { return 0 },
			maxB:  func(_ *simulation.Genome, _ *simulation.Parameters) byte { return 255 },
		},
		{
			label: "Metabolic",
			get:   func(g *simulation.Genome) byte { return g.MetabolicRate },
			set:   func(g *simulation.Genome, v byte) { g.MetabolicRate = v },
			minB:  func(_ *simulation.Genome, _ *simulation.Parameters) byte { return 0 },
			maxB:  func(_ *simulation.Genome, _ *simulation.Parameters) byte { return 255 },
		},
		{
			label: "Stomach",
			get:   func(g *simulation.Genome) byte { return g.StomachSize },
			set:   func(g *simulation.Genome, v byte) { g.StomachSize = v },
			minB:  func(_ *simulation.Genome, _ *simulation.Parameters) byte { return 0 },
			maxB:  func(_ *simulation.Genome, _ *simulation.Parameters) byte { return 255 },
		},
		{
			label: "Learn Rate",
			get:   func(g *simulation.Genome) byte { return g.Neuroplasticity },
			set:   func(g *simulation.Genome, v byte) { g.Neuroplasticity = v },
			minB:  func(_ *simulation.Genome, _ *simulation.Parameters) byte { return 0 },
			maxB:  func(_ *simulation.Genome, _ *simulation.Parameters) byte { return 255 },
		},
		{
			label: "Learn Thresh",
			get:   func(g *simulation.Genome) byte { return g.LearningThreshold },
			set:   func(g *simulation.Genome, v byte) { g.LearningThreshold = v },
			minB:  func(_ *simulation.Genome, _ *simulation.Parameters) byte { return 0 },
			maxB:  func(_ *simulation.Genome, _ *simulation.Parameters) byte { return 255 },
		},
	}
}

// ── Supporting types ──────────────────────────────────────────────────────────

type nnNodeRef struct {
	typ byte
	id  byte
}

type traitSlotHit struct {
	trackX, trackY, trackW float32
}

type geNodeHit struct {
	cx, cy float32
	typ    byte
	id     byte
}

type edgeMidPoint struct {
	cx, cy  float32
	geneIdx int
}

// ── GenomeEditor ──────────────────────────────────────────────────────────────

// GenomeEditor is the full-screen modal panel for creating and editing genomes.
type GenomeEditor struct {
	visible bool
	genome  *simulation.Genome
	params  *simulation.Parameters
	onSave  func(*simulation.Genome, string)

	// Name input
	name             string
	nameInputFocused bool
	nameInputBounds  [4]float32

	traitDefs []traitDef

	// Trait slider drag state
	draggingTrait int

	// NN interaction state
	pendingSrc  *nnNodeRef
	selEdgeIdx  int
	selNeuronID int // -1 = none

	// Geometry written during Draw, read during HandleInput
	panX, panY float32
	traitSlots []traitSlotHit
	nodeHits   []geNodeHit
	edgeMidPts []edgeMidPoint

	// Sensor / action filter (which nodes to display in the NN graph)
	sensorFilter      [simulation.SENSOR_COUNT]bool
	actionFilter      [simulation.ACTION_COUNT]bool
	filterSensorScroll int
	// Bounds for filter checkboxes (populated each frame by drawFilterSection)
	filterSensorBounds          [][4]float32 // visible sensor rows [x,y,w,h]
	filterActionBounds          [][4]float32 // all action rows [x,y,w,h]
	filterSensorUp              [4]float32
	filterSensorDown            [4]float32
	filterSensorToggleAllBounds [4]float32
	filterActionToggleAllBounds [4]float32

	// Draggable hidden neuron positions (screen-relative offsets from panX/panY)
	neuronPos        map[int][2]float32 // key = neuron id, value = [screenX, screenY]
	draggingNeuronID int
	neuronDragOfsX   float32
	neuronDragOfsY   float32

	// NN controls
	addNeuronBtn *components.Button
	remNeuronBtn *components.Button
	delNeuronBtn *components.Button
	wtTrackX     float32
	wtTrackY     float32
	wtTrackW     float32
	delEdgeBtn   *components.Button
	saveBtn      *components.Button
	cancelBtn    *components.Button
	closeBtn     *components.Button
	loadSavedBtn *components.Button

	onLoadSaved func() // called when user clicks "Load Genome" button

	font      *textv2.GoXFace
	smallFont *textv2.GoXFace

	draggingWeight bool
}

func newGenomeEditor(font, smallFont *textv2.GoXFace, onSave func(*simulation.Genome, string)) *GenomeEditor {
	e := &GenomeEditor{
		traitDefs:        makeTraitDefs(),
		draggingTrait:    -1,
		selEdgeIdx:       -1,
		selNeuronID:      -1,
		draggingNeuronID: -1,
		onSave:           onSave,
		font:             font,
		smallFont:        smallFont,
		neuronPos:        make(map[int][2]float32),
		closeBtn:         &components.Button{W: 24, H: 22, Label: "×", Color: ColorBtnClose, LabelColor: color.White, Font: font},
		cancelBtn:        &components.Button{W: 100, H: 26, Label: "Cancel", Color: ColorBtnCancel, LabelColor: color.White, Font: font},
		saveBtn:          &components.Button{W: 130, H: 26, Label: "Save Genome", Color: ColorBtnSave, LabelColor: color.White, Font: font},
		addNeuronBtn:     &components.Button{W: 32, H: 22, Label: "+N", Color: ColorBtnAddNeuron, LabelColor: color.White, Font: font},
		remNeuronBtn:     &components.Button{W: 32, H: 22, Label: "-N", Color: ColorBtnRemNeuron, LabelColor: color.White, Font: font},
		delNeuronBtn:     &components.Button{W: 76, H: 22, Label: "Del Neuron", Color: ColorBtnRemNeuron, LabelColor: color.White, Font: font},
		delEdgeBtn:       &components.Button{W: 76, H: 22, Label: "Del Edge", Color: ColorBtnDelEdge, LabelColor: color.White, Font: font},
		loadSavedBtn:     &components.Button{W: 130, H: 26, Label: "Load Genome", Color: ColorBtnEdit, LabelColor: color.White, Font: font},
	}
	e.initFilters()
	return e
}

func (e *GenomeEditor) initFilters() {
	for i := range e.sensorFilter {
		e.sensorFilter[i] = false
	}
	for i := range e.actionFilter {
		e.actionFilter[i] = false
	}
}

// LoadGenome replaces the current genome without closing the editor.
func (e *GenomeEditor) LoadGenome(g *simulation.Genome) {
	e.genome = g.Copy()
	e.draggingTrait = -1
	e.pendingSrc = nil
	e.selEdgeIdx = -1
	e.selNeuronID = -1
	e.draggingWeight = false
	e.draggingNeuronID = -1
	e.neuronPos = make(map[int][2]float32)
	e.initFilters()
}

// Open loads a genome for editing. Pass nil to start with a fresh random one.
func (e *GenomeEditor) Open(g *simulation.Genome, p *simulation.Parameters) {
	e.params = p
	if g != nil {
		e.genome = g.Copy()
	} else {
		e.genome = &simulation.Genome{
			Brain:            make([]simulation.Gene, 0),
			SynapticDensity:  0,
			CognitiveBreadth: 0,
			BodyMass:         byte(15),
			SurvivalMass:     byte(10),
			ReproductionType: 0,
			MassSplitRatio:   127,
			MetabolicRate:    127,
			VisionRadius:     127,
			FieldOfView:      127,
			OscPeriod:        127,
			JuvenilePeriod:   127,
			MutationRate:     20,
			Responsiveness:   127,
			StomachSize:      127,
			Neuroplasticity:  127,
			LearningThreshold: 127,
		}
	}
	e.draggingTrait = -1
	e.pendingSrc = nil
	e.selEdgeIdx = -1
	e.selNeuronID = -1
	e.draggingWeight = false
	e.draggingNeuronID = -1
	e.neuronPos = make(map[int][2]float32)
	e.name = ""
	e.nameInputFocused = false
	e.filterSensorScroll = 0
	e.initFilters()
	e.visible = true
}

// ── Input handling ────────────────────────────────────────────────────────────

func (e *GenomeEditor) HandleInput(mx, my int) bool {
	if !e.visible {
		return false
	}
	fx, fy := float32(mx), float32(my)

	if e.closeBtn.IsClicked(mx, my) || e.cancelBtn.IsClicked(mx, my) {
		e.visible = false
		return true
	}

	if e.saveBtn.IsClicked(mx, my) {
		e.genome.SynapticDensity = byte(len(e.genome.Brain))
		if e.onSave != nil {
			e.onSave(e.genome, e.name)
		}
		e.visible = false
		return true
	}

	if e.loadSavedBtn.IsClicked(mx, my) {
		if e.onLoadSaved != nil {
			e.onLoadSaved()
		}
		return true
	}

	if inGeRect(fx, fy, e.nameInputBounds) {
		e.nameInputFocused = true
		return true
	}
	e.nameInputFocused = false

	if e.addNeuronBtn.IsClicked(mx, my) {
		if e.genome.CognitiveBreadth < e.params.Neurology.MaxSynapticDensity {
			e.genome.CognitiveBreadth++
			e.selNeuronID = -1
		}
		return true
	}

	if e.remNeuronBtn.IsClicked(mx, my) {
		if e.genome.CognitiveBreadth > e.params.Neurology.MinSynapticDensity {
			e.deleteNeuron(int(e.genome.CognitiveBreadth) - 1)
		}
		return true
	}

	if e.selNeuronID >= 0 && e.delNeuronBtn.IsClicked(mx, my) {
		e.deleteNeuron(e.selNeuronID)
		return true
	}

	if e.selEdgeIdx >= 0 && e.delEdgeBtn.IsClicked(mx, my) {
		idx := e.selEdgeIdx
		e.genome.Brain = append(e.genome.Brain[:idx], e.genome.Brain[idx+1:]...)
		e.selEdgeIdx = -1
		return true
	}

	// Weight slider click
	if e.selEdgeIdx >= 0 && e.selEdgeIdx < len(e.genome.Brain) {
		if fy >= e.wtTrackY-4 && fy <= e.wtTrackY+14 &&
			fx >= e.wtTrackX && fx <= e.wtTrackX+e.wtTrackW {
			e.draggingWeight = true
			e.applyWeightDrag(mx)
			return true
		}
	}

	// Toggle-all sensor checkboxes
	if inGeRect(fx, fy, e.filterSensorToggleAllBounds) {
		allOn := true
		for _, v := range e.sensorFilter {
			if !v {
				allOn = false
				break
			}
		}
		for i := range e.sensorFilter {
			e.sensorFilter[i] = !allOn
		}
		return true
	}
	// Toggle-all action checkboxes
	if inGeRect(fx, fy, e.filterActionToggleAllBounds) {
		allOn := true
		for _, v := range e.actionFilter {
			if !v {
				allOn = false
				break
			}
		}
		for i := range e.actionFilter {
			e.actionFilter[i] = !allOn
		}
		return true
	}

	// Sensor filter scroll
	if inGeRect(fx, fy, e.filterSensorUp) {
		if e.filterSensorScroll > 0 {
			e.filterSensorScroll--
		}
		return true
	}
	if inGeRect(fx, fy, e.filterSensorDown) {
		max := int(simulation.SENSOR_COUNT) - geFilterMaxSens
		if max < 0 {
			max = 0
		}
		if e.filterSensorScroll < max {
			e.filterSensorScroll++
		}
		return true
	}
	// Sensor filter checkboxes
	for i, b := range e.filterSensorBounds {
		if inGeRect(fx, fy, b) {
			sid := e.filterSensorScroll + i
			if sid < int(simulation.SENSOR_COUNT) {
				e.sensorFilter[sid] = !e.sensorFilter[sid]
			}
			return true
		}
	}
	// Action filter checkboxes
	for i, b := range e.filterActionBounds {
		if inGeRect(fx, fy, b) {
			if i < int(simulation.ACTION_COUNT) {
				e.actionFilter[i] = !e.actionFilter[i]
			}
			return true
		}
	}

	// Trait slider / toggle click
	for i, slot := range e.traitSlots {
		if i >= len(e.traitDefs) {
			break
		}
		td := e.traitDefs[i]
		rowH := float32(geTraitRowH)
		if fy >= slot.trackY-4 && fy <= slot.trackY+rowH &&
			fx >= slot.trackX && fx <= slot.trackX+slot.trackW+50 {
			if td.isToggle {
				cur := td.get(e.genome)
				if cur == 0 {
					td.set(e.genome, 1)
				} else {
					td.set(e.genome, 0)
				}
			} else {
				e.draggingTrait = i
				e.applyTraitDrag(i, mx)
			}
			return true
		}
	}

	// Edge midpoint selection
	for _, em := range e.edgeMidPts {
		dx, dy := fx-em.cx, fy-em.cy
		if dx*dx+dy*dy <= 10*10 {
			e.selEdgeIdx = em.geneIdx
			e.pendingSrc = nil
			return true
		}
	}

	// NN node click — start or complete a connection
	for _, nh := range e.nodeHits {
		dx, dy := fx-nh.cx, fy-nh.cy
		if dx*dx+dy*dy <= 12*12 {
			if e.pendingSrc == nil {
				if nh.typ == simulation.SENSOR || nh.typ == simulation.NEURON {
					ref := nnNodeRef{typ: nh.typ, id: nh.id}
					e.pendingSrc = &ref
					e.selEdgeIdx = -1
				}
			} else {
				if nh.typ == simulation.NEURON || nh.typ == simulation.ACTION {
					e.genome.Brain = append(e.genome.Brain, simulation.Gene{
						SourceType: e.pendingSrc.typ,
						SourceID:   e.pendingSrc.id,
						SinkType:   nh.typ,
						SinkID:     nh.id,
						Weight:     128,
					})
					e.genome.SynapticDensity = byte(len(e.genome.Brain))
				}
				e.pendingSrc = nil
			}
			if nh.typ == simulation.NEURON {
				e.selNeuronID = int(nh.id)
			} else {
				e.selNeuronID = -1
			}
			return true
		}
	}

	e.selEdgeIdx = -1
	e.selNeuronID = -1
	e.pendingSrc = nil
	return fx >= e.panX && fx <= e.panX+gePanW && fy >= e.panY && fy <= e.panY+gePanH
}

func (e *GenomeEditor) HandleKeyInput() {
	if !e.visible || !e.nameInputFocused {
		return
	}
	runes := ebiten.AppendInputChars([]rune(e.name))
	e.name = string(runes)
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len([]rune(e.name)) > 0 {
		r := []rune(e.name)
		e.name = string(r[:len(r)-1])
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		e.nameInputFocused = false
	}
}

func (e *GenomeEditor) HandleDrag(mx, my int) {
	if !e.visible {
		return
	}
	if e.draggingTrait >= 0 {
		e.applyTraitDrag(e.draggingTrait, mx)
	}
	if e.draggingWeight && e.selEdgeIdx >= 0 && e.selEdgeIdx < len(e.genome.Brain) {
		e.applyWeightDrag(mx)
	}
}

func (e *GenomeEditor) HandleRelease() {
	e.draggingTrait = -1
	e.draggingWeight = false
}

func (e *GenomeEditor) HandleRightClick(mx, my int) bool {
	if !e.visible {
		return false
	}
	fx, fy := float32(mx), float32(my)
	for _, nh := range e.nodeHits {
		if nh.typ != simulation.NEURON {
			continue
		}
		dx, dy := fx-nh.cx, fy-nh.cy
		if dx*dx+dy*dy <= 14*14 {
			e.draggingNeuronID = int(nh.id)
			e.neuronDragOfsX = nh.cx - fx
			e.neuronDragOfsY = nh.cy - fy
			return true
		}
	}
	return false
}

func (e *GenomeEditor) HandleRightDrag(mx, my int) {
	if !e.visible || e.draggingNeuronID < 0 {
		return
	}
	e.neuronPos[e.draggingNeuronID] = [2]float32{
		float32(mx) + e.neuronDragOfsX,
		float32(my) + e.neuronDragOfsY,
	}
}

func (e *GenomeEditor) HandleRightRelease() {
	e.draggingNeuronID = -1
}

func (e *GenomeEditor) applyTraitDrag(idx, mx int) {
	if idx < 0 || idx >= len(e.traitSlots) || idx >= len(e.traitDefs) {
		return
	}
	td := e.traitDefs[idx]
	if td.isToggle {
		return
	}
	slot := e.traitSlots[idx]
	lo := float64(td.minB(e.genome, e.params))
	hi := float64(td.maxB(e.genome, e.params))
	t := (float64(mx) - float64(slot.trackX)) / float64(slot.trackW)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	v := byte(math.Round(lo + t*(hi-lo)))
	if v < byte(lo) {
		v = byte(lo)
	}
	if v > byte(hi) {
		v = byte(hi)
	}
	td.set(e.genome, v)
}

func (e *GenomeEditor) deleteNeuron(k int) {
	nc := int(e.genome.CognitiveBreadth)
	if nc == 0 || k < 0 || k >= nc {
		return
	}
	var nb []simulation.Gene
	for _, gene := range e.genome.Brain {
		srcEff := int(gene.SourceID) % nc
		sinkEff := int(gene.SinkID) % nc
		if (gene.SourceType == simulation.NEURON && srcEff == k) ||
			(gene.SinkType == simulation.NEURON && sinkEff == k) {
			continue
		}
		// Renumber: effective IDs above k shift down by 1
		if gene.SourceType == simulation.NEURON && srcEff > k {
			gene.SourceID = byte(srcEff - 1)
		}
		if gene.SinkType == simulation.NEURON && sinkEff > k {
			gene.SinkID = byte(sinkEff - 1)
		}
		nb = append(nb, gene)
	}
	e.genome.Brain = nb
	e.genome.CognitiveBreadth--
	// Shift neuron positions down for IDs above k
	newPos := make(map[int][2]float32, len(e.neuronPos))
	for id, pos := range e.neuronPos {
		if id == k {
			continue
		}
		if id > k {
			newPos[id-1] = pos
		} else {
			newPos[id] = pos
		}
	}
	e.neuronPos = newPos
	e.selEdgeIdx = -1
	e.selNeuronID = -1
	e.pendingSrc = nil
}

func (e *GenomeEditor) applyWeightDrag(mx int) {
	if e.selEdgeIdx < 0 || e.selEdgeIdx >= len(e.genome.Brain) {
		return
	}
	t := (float64(mx) - float64(e.wtTrackX)) / float64(e.wtTrackW)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	e.genome.Brain[e.selEdgeIdx].Weight = byte(t * 255)
}

// ── Drawing ───────────────────────────────────────────────────────────────────

func (e *GenomeEditor) Draw(screen *ebiten.Image, fnt *textv2.GoXFace) {
	if !e.visible {
		return
	}

	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	e.panX = float32(sw/2 - gePanW/2)
	e.panY = float32(sh/2 - gePanH/2)
	px, py := e.panX, e.panY

	vector.FillRect(screen, px, py, gePanW, gePanH, ColorModalBG, false)
	vector.StrokeRect(screen, px, py, gePanW, gePanH, 2, ColorModalBorder, false)

	vector.FillRect(screen, px, py, gePanW, geTitleH, ColorModalTitleBar, false)

	m := fnt.Metrics()
	glyphH := float32(m.HAscent + m.HDescent)
	titleY := int(py) + int((geTitleH-glyphH)/2)
	drawText(screen, "GENOME EDITOR", fnt, int(px)+gePad, titleY, ColorModalTitle)

	cbx := px + gePanW - 28
	cby := py + 3
	e.closeBtn.Draw(screen, cbx, cby)

	// Vertical separator between traits/filter and NN section
	sepX := px + geNNSectionX - 8
	vector.FillRect(screen, sepX, py+geTitleH, 2, gePanH-geTitleH-geFooterH, ColorSeparator, false)

	e.drawTraits(screen, fnt)
	e.drawFilterSection(screen)
	e.drawNNSection(screen, fnt)

	// Footer
	footerY := py + gePanH - geFooterH
	vector.FillRect(screen, px, footerY, gePanW, geFooterH, ColorFooterBG, false)

	// Row 1 — Name input (use font metrics for vertical centering)
	nameLabelX := px + gePad
	nameInputX := nameLabelX + 52
	nameInputY := footerY + 4
	nameInputW := float32(gePanW) - 52 - gePad*3
	nameInputH := glyphH + 8
	e.nameInputBounds = [4]float32{nameInputX, nameInputY, nameInputW, nameInputH}

	labelTextY := int(nameInputY) + int((nameInputH-glyphH)/2)
	drawText(screen, "Name:", fnt, int(nameLabelX), labelTextY, ColorLabelSecondary)

	borderClr := ColorInputBorder
	if e.nameInputFocused {
		borderClr = ColorInputBorderFocused
	}
	vector.FillRect(screen, nameInputX, nameInputY, nameInputW, nameInputH, ColorInputBG, false)
	vector.StrokeRect(screen, nameInputX, nameInputY, nameInputW, nameInputH, 1, borderClr, false)

	displayName := e.name
	if e.nameInputFocused {
		displayName += "|"
	} else if e.name == "" {
		displayName = "(optional — leave blank for auto name)"
	}
	var nameClr color.Color = color.White
	if e.name == "" && !e.nameInputFocused {
		nameClr = ColorInputPlaceholder
	}
	inputTextY := int(nameInputY) + int((nameInputH-glyphH)/2)
	drawText(screen, displayName, fnt, int(nameInputX)+6, inputTextY, nameClr)

	// Row 2 — Cancel / Load / Save buttons (anchored from footer bottom with gePad clearance)
	btnRowY := footerY + float32(geFooterH) - float32(e.saveBtn.H) - gePad
	gap := float32(10)
	totalBtnW := e.cancelBtn.W + e.loadSavedBtn.W + e.saveBtn.W + gap*2
	bStartX := px + (gePanW-totalBtnW)/2
	e.cancelBtn.Draw(screen, bStartX, btnRowY)
	e.loadSavedBtn.Draw(screen, bStartX+e.cancelBtn.W+gap, btnRowY)
	e.saveBtn.Draw(screen, bStartX+e.cancelBtn.W+e.loadSavedBtn.W+gap*2, btnRowY)
}

// drawTraits renders the left-column trait sliders using font-metric positioning.
func (e *GenomeEditor) drawTraits(screen *ebiten.Image, fnt *textv2.GoXFace) {
	px, py := e.panX, e.panY
	secX := px + gePad
	secY := py + geTitleH + gePad

	m := fnt.Metrics()
	glyphH := float32(m.HAscent + m.HDescent)

	sectionLabelY := int(secY) + int((float32(geTraitRowH)-glyphH)/2)
	drawText(screen, "TRAITS", fnt, int(secX), sectionLabelY, ColorLabelMuted)

	e.traitSlots = e.traitSlots[:0]
	rowY := secY + float32(geTraitRowH)

	for i, td := range e.traitDefs {
		ry := rowY + float32(i)*float32(geTraitRowH)
		// Vertically centre the label text within the row height.
		textY := int(ry) + int((float32(geTraitRowH)-glyphH)/2)
		drawText(screen, td.label, fnt, int(secX), textY, ColorLabelSecondary)

		trackX := secX + geTraitTrackX0
		trackY := ry + (float32(geTraitRowH)-6)/2 // centre 6px track within row
		e.traitSlots = append(e.traitSlots, traitSlotHit{trackX: trackX, trackY: trackY, trackW: geTraitTrackW})

		if td.isToggle {
			cur := td.get(e.genome)
			lbl := "Asexual"
			btnClr := ColorBtnAsexual
			if cur == 1 {
				lbl = "Sexual"
				btnClr = ColorBtnSexual
			}
			toggleH := float32(geTraitRowH) - 4
			vector.FillRect(screen, trackX, ry+2, geTraitTrackW+44, toggleH, btnClr, false)
			toggleTextY := int(ry+2) + int((toggleH-glyphH)/2)
			drawText(screen, lbl, fnt, int(trackX)+10, toggleTextY, color.White)
			continue
		}

		lo := float64(0)
		hi := float64(255)
		if td.minB != nil {
			lo = float64(td.minB(e.genome, e.params))
		}
		if td.maxB != nil {
			hi = float64(td.maxB(e.genome, e.params))
		}
		cur := float64(td.get(e.genome))
		t := float32(0)
		if hi > lo {
			t = float32((cur - lo) / (hi - lo))
			if t < 0 {
				t = 0
			}
			if t > 1 {
				t = 1
			}
		}

		vector.FillRect(screen, trackX, trackY, geTraitTrackW, 6, ColorTraitTrackBG, false)
		vector.FillRect(screen, trackX, trackY, float32(geTraitTrackW)*t, 6, ColorTraitTrackFill, false)
		knobX := trackX + float32(geTraitTrackW)*t - 3
		vector.FillRect(screen, knobX, trackY-2, 6, 10, ColorTraitKnob, false)
		valTextY := int(ry) + int((float32(geTraitRowH)-glyphH)/2)
		drawText(screen, fmt.Sprintf("%d", byte(cur)), fnt, int(trackX)+geTraitTrackW+6, valTextY, ColorTraitValue)
	}
}

// drawFilterSection renders the sensor/action filter checkboxes below the traits.
func (e *GenomeEditor) drawFilterSection(screen *ebiten.Image) {
	if e.smallFont == nil {
		return
	}
	fnt := e.smallFont
	px, py := e.panX, e.panY
	secX := px + gePad
	// Section starts just below the traits.
	traitsBottom := py + geTitleH + gePad + float32(geTraitRowH)*(1+float32(len(e.traitDefs)))
	secY := traitsBottom + 6

	m := fnt.Metrics()
	glyphH := float32(m.HAscent + m.HDescent)

	// Section header
	drawText(screen, "FILTER NODES", fnt, int(secX), int(secY), ColorLabelMuted)
	secY += glyphH + 4

	// Separator line
	vector.FillRect(screen, secX, secY, float32(geTraitW-gePad), 1, ColorSeparator, false)
	secY += 3

	colW := float32(geTraitW-gePad) / 2 // half for sensors, half for actions
	sensX := secX
	actX := secX + colW + 2

	// Sub-headers
	drawText(screen, "Sensors", fnt, int(sensX), int(secY), ColorSensorHdr)
	drawText(screen, "Actions", fnt, int(actX), int(secY), ColorActionHdr)
	secY += glyphH + 2

	// Scroll buttons for sensors (above the list)
	btnSz := glyphH + 2
	upBtnBounds := [4]float32{sensX, secY, btnSz, btnSz}
	downBtnBounds := [4]float32{sensX + btnSz + 2, secY, btnSz, btnSz}
	e.filterSensorUp = upBtnBounds
	e.filterSensorDown = downBtnBounds

	upClr := ColorArrowDisabled
	if e.filterSensorScroll > 0 {
		upClr = ColorArrowEnabled
	}
	downClr := ColorArrowDisabled
	if e.filterSensorScroll < int(simulation.SENSOR_COUNT)-geFilterMaxSens {
		downClr = ColorArrowEnabled
	}
	vector.FillRect(screen, upBtnBounds[0], upBtnBounds[1], upBtnBounds[2], upBtnBounds[3], upClr, false)
	drawText(screen, "▲", fnt, int(upBtnBounds[0])+2, int(upBtnBounds[1]), color.White)
	vector.FillRect(screen, downBtnBounds[0], downBtnBounds[1], downBtnBounds[2], downBtnBounds[3], downClr, false)
	drawText(screen, "▼", fnt, int(downBtnBounds[0])+2, int(downBtnBounds[1]), color.White)

	// Toggle-all buttons
	toggleAllSX := sensX + btnSz*2 + 6
	vector.FillRect(screen, toggleAllSX, secY, 30, btnSz, ColorArrowEnabled, false)
	drawText(screen, "All", fnt, int(toggleAllSX)+4, int(secY), color.White)
	e.filterSensorToggleAllBounds = [4]float32{toggleAllSX, secY, 30, btnSz}
	e.filterSensorBounds = e.filterSensorBounds[:0]

	toggleAllAX := actX
	vector.FillRect(screen, toggleAllAX, secY, 30, btnSz, ColorArrowEnabled, false)
	drawText(screen, "All", fnt, int(toggleAllAX)+4, int(secY), color.White)
	e.filterActionToggleAllBounds = [4]float32{toggleAllAX, secY, 30, btnSz}
	e.filterActionBounds = e.filterActionBounds[:0]

	listY := secY + btnSz + 2

	// Sensor checkboxes (scrollable)
	const checkSz = float32(9)
	e.filterSensorBounds = e.filterSensorBounds[:0]
	for i := 0; i < geFilterMaxSens; i++ {
		sid := e.filterSensorScroll + i
		if sid >= int(simulation.SENSOR_COUNT) {
			break
		}
		rowY := listY + float32(i)*geFilterRowH
		checked := e.sensorFilter[sid]
		// Checkbox box
		cbClr := color.RGBA{60, 60, 80, 255}
		if checked {
			cbClr = color.RGBA{80, 180, 80, 255}
		}
		vector.FillRect(screen, sensX, rowY+(geFilterRowH-checkSz)/2, checkSz, checkSz, cbClr, false)
		// Label (truncated to column width)
		lbl := geTruncate(nnSensorName(byte(sid)), colW-checkSz-4, fnt)
		textY := int(rowY) + int((geFilterRowH-glyphH)/2)
		clr := color.RGBA{180, 180, 180, 255}
		if checked {
			clr = color.RGBA{220, 220, 220, 255}
		}
		drawText(screen, lbl, fnt, int(sensX)+int(checkSz)+3, textY, clr)
		e.filterSensorBounds = append(e.filterSensorBounds, [4]float32{sensX, rowY, colW - 2, geFilterRowH})
	}

	// Action checkboxes (full list — 11 items always fits)
	e.filterActionBounds = e.filterActionBounds[:0]
	for i := 0; i < int(simulation.ACTION_COUNT); i++ {
		rowY := listY + float32(i)*geFilterRowH
		// Don't draw past the separator
		if rowY+geFilterRowH > py+float32(gePanH-geFooterH-gePad) {
			break
		}
		checked := e.actionFilter[i]
		cbClr := color.RGBA{60, 60, 80, 255}
		if checked {
			cbClr = color.RGBA{180, 100, 80, 255}
		}
		vector.FillRect(screen, actX, rowY+(geFilterRowH-checkSz)/2, checkSz, checkSz, cbClr, false)
		lbl := geTruncate(nnActionName(byte(i)), colW-checkSz-4, fnt)
		textY := int(rowY) + int((geFilterRowH-glyphH)/2)
		clr := color.RGBA{180, 180, 180, 255}
		if checked {
			clr = color.RGBA{220, 220, 220, 255}
		}
		drawText(screen, lbl, fnt, int(actX)+int(checkSz)+3, textY, clr)
		e.filterActionBounds = append(e.filterActionBounds, [4]float32{actX, rowY, colW - 2, geFilterRowH})
	}
}

// drawNNSection renders the right-column neural-network editor.
func (e *GenomeEditor) drawNNSection(screen *ebiten.Image, fnt *textv2.GoXFace) {
	px, py := e.panX, e.panY
	nnX := px + geNNSectionX
	nnY := py + geTitleH + gePad

	m := fnt.Metrics()
	glyphH := float32(m.HAscent + m.HDescent)

	drawText(screen, "NEURAL NETWORK", fnt, int(nnX), int(nnY)+int((float32(geNNSubtitleH)-glyphH)/2), ColorLabelMuted)

	nodeAreaY := nnY + geNNSubtitleH + gePad
	ctrlY := py + gePanH - geFooterH - gePad - geNNControlH
	nodeAreaH := ctrlY - nodeAreaY - gePad

	// Build visible sensor/action lists from filter state
	numN := int(e.genome.CognitiveBreadth)

	var visibleSensors []int
	for i := 0; i < int(simulation.SENSOR_COUNT); i++ {
		if e.sensorFilter[i] {
			visibleSensors = append(visibleSensors, i)
		}
	}
	var visibleActions []int
	for i := 0; i < int(simulation.ACTION_COUNT); i++ {
		if e.actionFilter[i] {
			visibleActions = append(visibleActions, i)
		}
	}

	numS := len(visibleSensors)
	numA := len(visibleActions)

	stepS := nodeAreaH
	if numS > 0 {
		stepS = nodeAreaH / float32(numS)
	}
	stepA := nodeAreaH
	if numA > 0 {
		stepA = nodeAreaH / float32(numA)
	}
	stepN := nodeAreaH
	if numN > 0 {
		stepN = nodeAreaH / float32(numN)
	}

	sAbsX := nnX + geNNSensorX
	nAbsX := nnX + geNNNeuronX
	aAbsX := nnX + geNNActionX

	// NN right boundary for text clipping
	nnRight := nnX + geNNW

	sY := make([]float32, numS)
	for i := range sY {
		sY[i] = nodeAreaY + stepS*float32(i) + stepS/2
	}
	aY := make([]float32, numA)
	for i := range aY {
		aY[i] = nodeAreaY + stepA*float32(i) + stepA/2
	}
	// Default neuron Y positions
	defaultNY := make([]float32, numN)
	for i := range defaultNY {
		defaultNY[i] = nodeAreaY + stepN*float32(i) + stepN/2
	}
	// Resolve neuron positions (user overrides or defaults)
	nX := make([]float32, numN)
	nYFinal := make([]float32, numN)
	for i := range nX {
		if pos, ok := e.neuronPos[i]; ok {
			nX[i] = pos[0]
			nYFinal[i] = pos[1]
		} else {
			nX[i] = nAbsX
			nYFinal[i] = defaultNY[i]
		}
	}

	// Map from raw gene sensor/action IDs to visible indices
	sensorToIdx := make(map[int]int, numS)
	for i, sid := range visibleSensors {
		sensorToIdx[sid] = i
	}
	actionToIdx := make(map[int]int, numA)
	for i, aid := range visibleActions {
		actionToIdx[aid] = i
	}

	effN := func(rawID byte) int {
		if numN == 0 {
			return 0
		}
		return int(rawID) % numN
	}

	// ── Edges ─────────────────────────────────────────────────────────────────
	e.edgeMidPts = e.edgeMidPts[:0]
	for gi, gene := range e.genome.Brain {
		var srcX, srcY, dstX, dstY float32
		valid := true

		switch gene.SourceType {
		case simulation.SENSOR:
			si, ok := sensorToIdx[int(gene.SourceID)%int(simulation.SENSOR_COUNT)]
			if !ok {
				valid = false
			} else {
				srcX, srcY = sAbsX, sY[si]
			}
		case simulation.NEURON:
			if numN == 0 {
				valid = false
			} else {
				ni := effN(gene.SourceID)
				srcX, srcY = nX[ni], nYFinal[ni]
			}
		default:
			valid = false
		}

		if valid {
			switch gene.SinkType {
			case simulation.NEURON:
				if numN == 0 {
					valid = false
				} else {
					ni := effN(gene.SinkID)
					dstX, dstY = nX[ni], nYFinal[ni]
				}
			case simulation.ACTION:
				ai, ok := actionToIdx[int(gene.SinkID)%int(simulation.ACTION_COUNT)]
				if !ok {
					valid = false
				} else {
					dstX, dstY = aAbsX, aY[ai]
				}
			default:
				valid = false
			}
		}

		if !valid {
			e.edgeMidPts = append(e.edgeMidPts, edgeMidPoint{geneIdx: gi})
			continue
		}

		isSelected := gi == e.selEdgeIdx
		ec := geEdgeColor(gene.WeightAsFloat32())
		if isSelected {
			ec = ColorNodeSelect
			ec.A = 230
		}

		// Self-loop
		if gene.SourceType == simulation.NEURON && gene.SinkType == simulation.NEURON &&
			effN(gene.SourceID) == effN(gene.SinkID) {
			ni := effN(gene.SourceID)
			lx, ly := nX[ni], nYFinal[ni]-geNodeR*2.8
			vector.StrokeCircle(screen, lx, ly, geNodeR*1.6, 1.5, ec, false)
			midX, midY := lx, ly-geNodeR*1.6
			e.edgeMidPts = append(e.edgeMidPts, edgeMidPoint{cx: midX, cy: midY, geneIdx: gi})
			dotClr := ec
			if isSelected {
				dotClr = ColorNodeSelect
			}
			vector.FillCircle(screen, midX, midY, geEdgeMidR, dotClr, false)
			continue
		}

		vector.StrokeLine(screen, srcX, srcY, dstX, dstY, 1.2, ec, false)
		midX, midY := (srcX+dstX)/2, (srcY+dstY)/2
		dotClr := ec
		if isSelected {
			dotClr = ColorNodeSelect
		}
		vector.FillCircle(screen, midX, midY, geEdgeMidR, dotClr, false)
		e.edgeMidPts = append(e.edgeMidPts, edgeMidPoint{cx: midX, cy: midY, geneIdx: gi})
	}

	// Pending-connection preview
	if e.pendingSrc != nil {
		cmx, cmy := ebiten.CursorPosition()
		var srcX, srcY float32
		switch e.pendingSrc.typ {
		case simulation.SENSOR:
			if si, ok := sensorToIdx[int(e.pendingSrc.id)]; ok {
				srcX, srcY = sAbsX, sY[si]
			}
		case simulation.NEURON:
			if numN > 0 {
				ni := effN(e.pendingSrc.id)
				srcX, srcY = nX[ni], nYFinal[ni]
			}
		}
		vector.StrokeLine(screen, srcX, srcY, float32(cmx), float32(cmy), 1.5, ColorPendingLine, false)
	}

	// ── Nodes ─────────────────────────────────────────────────────────────────
	e.nodeHits = e.nodeHits[:0]

	// Column headers
	drawText(screen, "Sensors", fnt, int(sAbsX)-32, int(nodeAreaY)-6, ColorSensorHdr)
	drawText(screen, "Neurons", fnt, int(nAbsX)-26, int(nodeAreaY)-6, ColorNeuronHdr)
	drawText(screen, "Actions", fnt, int(aAbsX)-10, int(nodeAreaY)-6, ColorActionHdr)

	// Max label width for sensor labels (must not go left of nnX)
	maxSensorLabelW := sAbsX - nnX - 10

	// Sensor nodes
	for i, sid := range visibleSensors {
		sy := sY[i]
		clr := ColorSensorNode
		if e.pendingSrc != nil && e.pendingSrc.typ == simulation.SENSOR && int(e.pendingSrc.id) == sid {
			clr = ColorNodeSelect
		}
		vector.FillCircle(screen, sAbsX, sy, geNodeR, clr, false)

		lbl := geTruncate(nnSensorName(byte(sid)), maxSensorLabelW, fnt)
		tw, _ := textv2.Measure(lbl, fnt, 0)
		tx := int(sAbsX - float32(tw) - 7)
		ty := int(sy - glyphH/2)
		drawText(screen, lbl, fnt, tx, ty, ColorSensorLabel)
		e.nodeHits = append(e.nodeHits, geNodeHit{cx: sAbsX, cy: sy, typ: simulation.SENSOR, id: byte(sid)})
	}

	// Neuron nodes (draggable)
	for i := 0; i < numN; i++ {
		nx, ny := nX[i], nYFinal[i]
		// Clamp to NN node area
		if nx < nnX+geNNSensorX+20 {
			nx = nnX + geNNSensorX + 20
		}
		if nx > nnRight-20 {
			nx = nnRight - 20
		}
		if ny < nodeAreaY {
			ny = nodeAreaY
		}
		if ny > ctrlY-10 {
			ny = ctrlY - 10
		}

		clr := ColorNeuronNode
		if e.pendingSrc != nil && e.pendingSrc.typ == simulation.NEURON && effN(e.pendingSrc.id) == i {
			clr = ColorNodeSelect
		}
		if e.draggingNeuronID == i {
			clr = ColorNodeSelect
		}
		vector.FillCircle(screen, nx, ny, geNodeR+1, clr, false)

		lbl := fmt.Sprintf("N%d", i)
		tw, _ := textv2.Measure(lbl, fnt, 0)
		// Place label to the right; if it'd overflow, place left
		lblX := nx + geNodeR + 3
		if lblX+float32(tw) > nnRight-2 {
			lblX = nx - float32(tw) - geNodeR - 3
		}
		ty := int(ny - glyphH/2)
		drawText(screen, lbl, fnt, int(lblX), ty, ColorNeuronLabel)
		e.nodeHits = append(e.nodeHits, geNodeHit{cx: nx, cy: ny, typ: simulation.NEURON, id: byte(i)})
	}

	// Action nodes
	maxActionLabelW := nnRight - aAbsX - 10
	for i, aid := range visibleActions {
		ay := aY[i]
		vector.FillCircle(screen, aAbsX, ay, geNodeR, ColorActionNode, false)

		lbl := geTruncate(nnActionName(byte(aid)), maxActionLabelW, fnt)
		ty := int(ay - glyphH/2)
		drawText(screen, lbl, fnt, int(aAbsX)+9, ty, ColorActionLabel)
		e.nodeHits = append(e.nodeHits, geNodeHit{cx: aAbsX, cy: ay, typ: simulation.ACTION, id: byte(aid)})
	}

	// ── Controls strip ────────────────────────────────────────────────────────
	vector.FillRect(screen, nnX, ctrlY, geNNW, geNNControlH, ColorCtrlStripBG, false)
	vector.StrokeRect(screen, nnX, ctrlY, geNNW, geNNControlH, 1, ColorCtrlStripBorder, false)

	bnx := nnX + gePad
	bny := ctrlY + gePad
	e.addNeuronBtn.Draw(screen, bnx, bny)
	e.remNeuronBtn.Draw(screen, bnx+36, bny)

	neuronInfo := fmt.Sprintf("Neurons: %d/%d  (drag to reposition)", e.genome.CognitiveBreadth, e.params.Neurology.MaxSynapticDensity)
	neuronInfoY := int(bny) + int((22-glyphH)/2)
	drawText(screen, neuronInfo, fnt, int(bnx)+74, neuronInfoY, ColorWeightLabel)

	rightBtnX := nnX + geNNW - float32(e.delEdgeBtn.W) - gePad
	if e.selNeuronID >= 0 && e.selEdgeIdx >= 0 {
		e.delEdgeBtn.Draw(screen, rightBtnX, bny)
		e.delNeuronBtn.Draw(screen, rightBtnX-float32(e.delNeuronBtn.W)-6, bny)
	} else if e.selEdgeIdx >= 0 {
		e.delEdgeBtn.Draw(screen, rightBtnX, bny)
	} else if e.selNeuronID >= 0 {
		e.delNeuronBtn.Draw(screen, rightBtnX, bny)
	}

	row2Y := ctrlY + gePad + 28
	if e.selEdgeIdx >= 0 && e.selEdgeIdx < len(e.genome.Brain) {
		gene := e.genome.Brain[e.selEdgeIdx]
		w := gene.WeightAsFloat32()

		labelEndX := nnX + gePad + 54
		wtLblY := int(row2Y) + int((22-glyphH)/2)
		drawText(screen, "Weight:", fnt, int(nnX)+gePad, wtLblY, ColorWeightLabel)

		trkW := float32(geNNW) - (labelEndX - nnX) - 54
		e.wtTrackX = labelEndX
		e.wtTrackY = row2Y + 4
		e.wtTrackW = trkW

		vector.FillRect(screen, e.wtTrackX, e.wtTrackY, e.wtTrackW, 6, ColorTraitTrackBG, false)
		tFrac := (w + 1.0) / 2.0
		fillClr := ColorWeightPos
		if w < 0 {
			fillClr = ColorWeightNeg
		}
		vector.FillRect(screen, e.wtTrackX, e.wtTrackY, e.wtTrackW*tFrac, 6, fillClr, false)
		knobX := e.wtTrackX + e.wtTrackW*tFrac - 3
		vector.FillRect(screen, knobX, e.wtTrackY-2, 6, 10, ColorWeightKnob, false)
		valY := int(row2Y) + int((22-glyphH)/2)
		drawText(screen, fmt.Sprintf("%.2f", w), fnt,
			int(e.wtTrackX+e.wtTrackW)+6, valY, ColorWeightLabel)
	} else if e.pendingSrc != nil {
		var srcDesc string
		switch e.pendingSrc.typ {
		case simulation.SENSOR:
			srcDesc = "Sensor: " + nnSensorName(e.pendingSrc.id)
		case simulation.NEURON:
			srcDesc = fmt.Sprintf("Neuron: N%d", e.pendingSrc.id)
		}
		hintY := int(row2Y) + int((22-glyphH)/2)
		drawText(screen, "Click target to connect from "+srcDesc,
			fnt, int(nnX)+gePad, hintY, ColorPendingHint)
	} else {
		hintY := int(row2Y) + int((22-glyphH)/2)
		drawText(screen, "Left-click node to connect  |  Click edge dot to select  |  Right-click neuron to move",
			fnt, int(nnX)+gePad, hintY, ColorHintMuted)
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func inGeRect(px, py float32, b [4]float32) bool {
	return px >= b[0] && px <= b[0]+b[2] && py >= b[1] && py <= b[1]+b[3]
}

func geEdgeColor(w float32) color.RGBA {
	abs := w
	if abs < 0 {
		abs = -abs
	}
	if abs > 1 {
		abs = 1
	}
	if w >= 0 {
		return color.RGBA{0, uint8(abs*200 + 30), 0, 180}
	}
	return color.RGBA{uint8(abs*180 + 30), uint8(abs * 40), 0, 160}
}

// geTruncate trims s so its rendered width fits within maxW pixels, appending "…" if needed.
func geTruncate(s string, maxW float32, face *textv2.GoXFace) string {
	if face == nil || maxW <= 0 {
		return s
	}
	w, _ := textv2.Measure(s, face, 0)
	if float32(w) <= maxW {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 {
		runes = runes[:len(runes)-1]
		w, _ = textv2.Measure(string(runes)+"…", face, 0)
		if float32(w) <= maxW {
			return string(runes) + "…"
		}
	}
	return ""
}
