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
	gePanW         = 820
	gePanH         = 670 // +30 for the name-input row in the footer
	geTitleH       = 28
	geFooterH      = 66 // two rows: name input (30px) + buttons (36px)
	gePad          = 10
	geTraitW       = 240
	geNNSectionX   = gePad + geTraitW + 16         // 266 — start of NN section within panel
	geNNW          = gePanW - geNNSectionX - gePad // 544
	geNNSubtitleH  = 22
	geNNControlH   = 72 // height of the controls strip at the bottom of the NN section
	geTraitRowH    = 26
	geTraitLabelW  = 98
	geTraitTrackW  = 100
	geTraitTrackX0 = geTraitLabelW + 6 // x of track within a trait row

	// Node column offsets within the NN section
	geNNSensorX = 95
	geNNNeuronX = 295
	geNNActionX = 472

	geNodeR    float32 = 5
	geEdgeMidR float32 = 4
)

// ── Trait definitions ─────────────────────────────────────────────────────────

type traitDef struct {
	label    string
	get      func(*simulation.Genome) byte
	set      func(*simulation.Genome, byte)
	minB     func(*simulation.Genome, *simulation.Parameters) byte
	maxB     func(*simulation.Genome, *simulation.Parameters) byte
	isToggle bool // render as a flip-button rather than a slider
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
			minB:  func(_ *simulation.Genome, p *simulation.Parameters) byte { return 0 },
			maxB:  func(_ *simulation.Genome, p *simulation.Parameters) byte { return 255 },
		},
		{
			label: "Field of View",
			get:   func(g *simulation.Genome) byte { return g.FieldOfView },
			set:   func(g *simulation.Genome, v byte) { g.FieldOfView = v },
			minB:  func(_ *simulation.Genome, p *simulation.Parameters) byte { return 0 },
			maxB:  func(_ *simulation.Genome, p *simulation.Parameters) byte { return 255 },
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

// nnNodeRef identifies a clickable node in the NN graph.
type nnNodeRef struct {
	typ byte // simulation.SENSOR, NEURON, or ACTION
	id  byte
}

// traitSlotHit stores the track geometry of a rendered trait row.
type traitSlotHit struct {
	trackX, trackY, trackW float32
}

// geNodeHit is a clickable node drawn on the NN canvas.
type geNodeHit struct {
	cx, cy float32
	typ    byte
	id     byte
}

// edgeMidPoint is the selectable midpoint dot of a brain gene edge.
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
	onSave  func(*simulation.Genome, string) // genome + user-supplied name

	// Name input
	name             string
	nameInputFocused bool
	nameInputBounds  [4]float32

	traitDefs []traitDef

	// Trait slider drag state
	draggingTrait int // -1 = none

	// NN interaction state
	pendingSrc *nnNodeRef // non-nil while waiting for a sink node click
	selEdgeIdx int        // -1 = no selected edge

	// Geometry written during Draw, read during HandleInput
	panX, panY float32
	traitSlots []traitSlotHit
	nodeHits   []geNodeHit
	edgeMidPts []edgeMidPoint

	// NN controls
	addNeuronBtn *components.Button
	remNeuronBtn *components.Button
	wtTrackX     float32
	wtTrackY     float32
	wtTrackW     float32
	delEdgeBtn   *components.Button
	saveBtn      *components.Button
	cancelBtn    *components.Button
	closeBtn     *components.Button

	font           *textv2.GoXFace
	draggingWeight bool
}

func newGenomeEditor(font *textv2.GoXFace, onSave func(*simulation.Genome, string)) *GenomeEditor {
	return &GenomeEditor{
		traitDefs:     makeTraitDefs(),
		draggingTrait: -1,
		selEdgeIdx:    -1,
		onSave:        onSave,
		font:          font,
		closeBtn:      &components.Button{W: 24, H: 22, Label: "×", Color: ColorBtnClose, LabelColor: color.White, Font: font},
		cancelBtn:     &components.Button{W: 100, H: 26, Label: "Cancel", Color: ColorBtnCancel, LabelColor: color.White, Font: font},
		saveBtn:       &components.Button{W: 130, H: 26, Label: "Save Genome", Color: ColorBtnSave, LabelColor: color.White, Font: font},
		addNeuronBtn:  &components.Button{W: 32, H: 22, Label: "+N", Color: ColorBtnAddNeuron, LabelColor: color.White, Font: font},
		remNeuronBtn:  &components.Button{W: 32, H: 22, Label: "-N", Color: ColorBtnRemNeuron, LabelColor: color.White, Font: font},
		delEdgeBtn:    &components.Button{W: 76, H: 22, Label: "Del Edge", Color: ColorBtnDelEdge, LabelColor: color.White, Font: font},
	}
}

// Open loads a genome for editing. Pass nil to start with a fresh random one.
func (e *GenomeEditor) Open(g *simulation.Genome, p *simulation.Parameters) {
	e.params = p
	if g != nil {
		e.genome = g.Copy()
	} else {

		e.genome = &simulation.Genome{
			// Neural Blueprint: Zero edges, Zero internal neurons
			Brain:            make([]simulation.Gene, 0),
			SynapticDensity:  0,
			CognitiveBreadth: 0,

			// Physical Invariants
			BodyMass:    byte(15),
			SurvivalMass: byte(10),

			// Reproduction: Default to Asexual Circle (0)
			ReproductionType: 0,
			MassSplitRatio:   127, // 50% split

			// Mid-range defaults for all other traits (0-255 scale)
			MetabolicRate:     127,
			VisionRadius:     127,
			FieldOfView:       127,
			OscPeriod:         127,
			JuvenilePeriod:    127,
			MutationRate:      20, // Subtle evolution
			Responsiveness:    127,
			StomachSize:       127,
			Neuroplasticity:   127,
			LearningThreshold: 127,
		}
	}
	e.draggingTrait = -1
	e.pendingSrc = nil
	e.selEdgeIdx = -1
	e.draggingWeight = false
	e.name = ""
	e.nameInputFocused = false
	e.visible = true
}

// ── Input handling ────────────────────────────────────────────────────────────

// HandleInput processes a mouse click. Returns true if the editor consumed it.
func (e *GenomeEditor) HandleInput(mx, my int) bool {
	if !e.visible {
		return false
	}
	fx, fy := float32(mx), float32(my)

	// Close / cancel
	if e.closeBtn.IsClicked(mx, my) || e.cancelBtn.IsClicked(mx, my) {
		e.visible = false
		return true
	}

	// Save
	if e.saveBtn.IsClicked(mx, my) {
		e.genome.SynapticDensity = byte(len(e.genome.Brain))
		if e.onSave != nil {
			e.onSave(e.genome, e.name)
		}
		e.visible = false
		return true
	}

	// Name input focus
	if inGeRect(fx, fy, e.nameInputBounds) {
		e.nameInputFocused = true
		return true
	}
	e.nameInputFocused = false

	// Add neuron
	if e.addNeuronBtn.IsClicked(mx, my) {
		if e.genome.CognitiveBreadth < e.params.Neurology.MaxSynapticDensity {
			e.genome.CognitiveBreadth++
		}
		return true
	}

	// Remove last neuron and all genes referencing it
	if e.remNeuronBtn.IsClicked(mx, my) {
		if e.genome.CognitiveBreadth > e.params.Neurology.MinSynapticDensity {
			lastID := e.genome.CognitiveBreadth - 1
			nc := e.genome.CognitiveBreadth
			var nb []simulation.Gene
			for _, gene := range e.genome.Brain {
				skip := (gene.SourceType == simulation.NEURON && gene.SourceID%nc == lastID) ||
					(gene.SinkType == simulation.NEURON && gene.SinkID%nc == lastID)
				if !skip {
					nb = append(nb, gene)
				}
			}
			e.genome.Brain = nb
			e.genome.CognitiveBreadth--
			e.selEdgeIdx = -1
		}
		return true
	}

	// Delete selected edge
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
			return true
		}
	}

	// Click elsewhere: deselect
	e.selEdgeIdx = -1
	e.pendingSrc = nil
	// Block clicks that land inside the panel bounds
	return fx >= e.panX && fx <= e.panX+gePanW && fy >= e.panY && fy <= e.panY+gePanH
}

// HandleKeyInput processes keyboard input for the name text field.
// Call this each frame from Game.Update when the editor is visible.
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

// HandleDrag is called every frame while the mouse button is held.
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

// HandleRelease clears any active drag state.
func (e *GenomeEditor) HandleRelease() {
	e.draggingTrait = -1
	e.draggingWeight = false
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

// Draw renders the genome editor as a full-panel overlay. Call after everything else.
func (e *GenomeEditor) Draw(screen *ebiten.Image, fnt *textv2.GoXFace) {
	if !e.visible {
		return
	}

	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	e.panX = float32(sw/2 - gePanW/2)
	e.panY = float32(sh/2 - gePanH/2)
	px, py := e.panX, e.panY

	// Panel background + border
	vector.FillRect(screen, px, py, gePanW, gePanH, ColorModalBG, false)
	vector.StrokeRect(screen, px, py, gePanW, gePanH, 2, ColorModalBorder, false)

	// Title bar
	vector.FillRect(screen, px, py, gePanW, geTitleH, ColorModalTitleBar, false)
	drawText(screen, "GENOME EDITOR", fnt, int(px)+gePad, int(py)+18, ColorModalTitle)

	// Close [×]
	cbx := px + gePanW - 28
	cby := py + 3
	e.closeBtn.Draw(screen, cbx, cby)

	// Vertical separator
	sepX := px + geNNSectionX - 8
	vector.FillRect(screen, sepX, py+geTitleH, 2, gePanH-geTitleH-geFooterH, ColorSeparator, false)

	e.drawTraits(screen, fnt)
	e.drawNNSection(screen, fnt)

	// Footer
	footerY := py + gePanH - geFooterH
	vector.FillRect(screen, px, footerY, gePanW, geFooterH, ColorFooterBG, false)

	// Row 1 — Name input
	nameLabelX := px + gePad
	nameInputX := nameLabelX + 52
	nameInputY := footerY + 4
	nameInputW := float32(gePanW) - 52 - gePad*3
	nameInputH := float32(22)
	e.nameInputBounds = [4]float32{nameInputX, nameInputY, nameInputW, nameInputH}

	drawText(screen, "Name:", fnt, int(nameLabelX), int(nameInputY)+15, ColorLabelSecondary)
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
	drawText(screen, displayName, fnt, int(nameInputX)+6, int(nameInputY)+15, nameClr)

	// Row 2 — Cancel / Save buttons
	btnRowY := footerY + 32
	gap := float32(20)
	bStartX := px + (gePanW-e.cancelBtn.W-e.saveBtn.W-gap)/2
	e.cancelBtn.Draw(screen, bStartX, btnRowY)
	e.saveBtn.Draw(screen, bStartX+e.cancelBtn.W+gap, btnRowY)
}

// drawTraits renders the left-column trait sliders.
func (e *GenomeEditor) drawTraits(screen *ebiten.Image, fnt *textv2.GoXFace) {
	px, py := e.panX, e.panY
	secX := px + gePad
	secY := py + geTitleH + gePad

	drawText(screen, "TRAITS", fnt, int(secX), int(secY)+14, ColorLabelMuted)

	e.traitSlots = e.traitSlots[:0]
	rowY := secY + 22

	for i, td := range e.traitDefs {
		ry := rowY + float32(i)*geTraitRowH
		drawText(screen, td.label, fnt, int(secX), int(ry)+14, ColorLabelSecondary)

		trackX := secX + geTraitTrackX0
		trackY := ry + 5
		e.traitSlots = append(e.traitSlots, traitSlotHit{trackX: trackX, trackY: trackY, trackW: geTraitTrackW})

		if td.isToggle {
			cur := td.get(e.genome)
			lbl := "Asexual"
			btnClr := ColorBtnAsexual
			if cur == 1 {
				lbl = "Sexual"
				btnClr = ColorBtnSexual
			}
			vector.FillRect(screen, trackX, trackY-2, geTraitTrackW+44, 20, btnClr, false)
			drawText(screen, lbl, fnt, int(trackX)+10, int(trackY)+13, color.White)
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

		// Track background
		vector.FillRect(screen, trackX, trackY, geTraitTrackW, 6, ColorTraitTrackBG, false)
		// Fill
		vector.FillRect(screen, trackX, trackY, float32(geTraitTrackW)*t, 6, ColorTraitTrackFill, false)
		// Handle knob
		knobX := trackX + float32(geTraitTrackW)*t - 3
		vector.FillRect(screen, knobX, trackY-2, 6, 10, ColorTraitKnob, false)
		// Value
		drawText(screen, fmt.Sprintf("%d", byte(cur)), fnt, int(trackX)+geTraitTrackW+6, int(ry)+14, ColorTraitValue)
	}
}

// drawNNSection renders the right-column neural-network editor.
func (e *GenomeEditor) drawNNSection(screen *ebiten.Image, fnt *textv2.GoXFace) {
	px, py := e.panX, e.panY
	nnX := px + geNNSectionX
	nnY := py + geTitleH + gePad

	drawText(screen, "NEURAL NETWORK", fnt, int(nnX), int(nnY)+14, ColorLabelMuted)

	nodeAreaY := nnY + geNNSubtitleH + gePad
	ctrlY := py + gePanH - geFooterH - gePad - geNNControlH
	nodeAreaH := ctrlY - nodeAreaY - gePad

	numS := int(simulation.SENSOR_COUNT)
	numA := int(simulation.ACTION_COUNT)
	numN := int(e.genome.CognitiveBreadth)

	stepS := nodeAreaH / float32(numS)
	stepA := nodeAreaH / float32(numA)
	stepN := float32(0)
	if numN > 0 {
		stepN = nodeAreaH / float32(numN)
	}

	sAbsX := nnX + geNNSensorX
	nAbsX := nnX + geNNNeuronX
	aAbsX := nnX + geNNActionX

	sY := make([]float32, numS)
	for i := range sY {
		sY[i] = nodeAreaY + stepS*float32(i) + stepS/2
	}
	aY := make([]float32, numA)
	for i := range aY {
		aY[i] = nodeAreaY + stepA*float32(i) + stepA/2
	}
	nY := make([]float32, numN)
	for i := range nY {
		nY[i] = nodeAreaY + stepN*float32(i) + stepN/2
	}

	effN := func(rawID byte) int {
		if numN == 0 {
			return 0
		}
		return int(rawID) % numN
	}

	// ── Edges ────────────────────────────────────────────────────────────────
	e.edgeMidPts = e.edgeMidPts[:0]
	for gi, gene := range e.genome.Brain {
		var srcX, srcY, dstX, dstY float32
		valid := true

		switch gene.SourceType {
		case simulation.SENSOR:
			sid := int(gene.SourceID) % numS
			srcX, srcY = sAbsX, sY[sid]
		case simulation.NEURON:
			if numN == 0 {
				valid = false
				break
			}
			srcX, srcY = nAbsX, nY[effN(gene.SourceID)]
		default:
			valid = false
		}

		if valid {
			switch gene.SinkType {
			case simulation.NEURON:
				if numN == 0 {
					valid = false
				} else {
					dstX, dstY = nAbsX, nY[effN(gene.SinkID)]
				}
			case simulation.ACTION:
				aid := int(gene.SinkID) % numA
				dstX, dstY = aAbsX, aY[aid]
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

		// Self-loop: draw a small circle above the neuron node
		if gene.SourceType == simulation.NEURON && gene.SinkType == simulation.NEURON &&
			effN(gene.SourceID) == effN(gene.SinkID) {
			lx := nAbsX
			ly := nY[effN(gene.SourceID)] - geNodeR*2.8
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

	// Pending-connection preview line to cursor
	if e.pendingSrc != nil {
		cmx, cmy := ebiten.CursorPosition()
		var srcX, srcY float32
		switch e.pendingSrc.typ {
		case simulation.SENSOR:
			srcX, srcY = sAbsX, sY[int(e.pendingSrc.id)%numS]
		case simulation.NEURON:
			if numN > 0 {
				srcX, srcY = nAbsX, nY[effN(e.pendingSrc.id)]
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

	// Sensor nodes
	for i := 0; i < numS; i++ {
		sy := sY[i]
		clr := ColorSensorNode
		if e.pendingSrc != nil && e.pendingSrc.typ == simulation.SENSOR && int(e.pendingSrc.id) == i {
			clr = ColorNodeSelect
		}
		vector.FillCircle(screen, sAbsX, sy, geNodeR, clr, false)

		lbl := nnSensorName(byte(i))

		metrics := fnt.Metrics()
		textHeight := float32(metrics.HAscent + metrics.HDescent)

		tw, _ := textv2.Measure(lbl, fnt, 0)
		lblW := float32(tw)

		tx := int(sAbsX - lblW - 7)
		ty := int(sy - textHeight/2)

		drawText(screen, lbl, fnt, tx, ty, ColorSensorLabel)
		e.nodeHits = append(e.nodeHits, geNodeHit{cx: sAbsX, cy: sy, typ: simulation.SENSOR, id: byte(i)})
	}

	// Neuron nodes
	for i := 0; i < numN; i++ {
		ny := nY[i]
		clr := ColorNeuronNode
		if e.pendingSrc != nil && e.pendingSrc.typ == simulation.NEURON && effN(e.pendingSrc.id) == i {
			clr = ColorNodeSelect
		}
		vector.FillCircle(screen, nAbsX, ny, geNodeR+1, clr, false)

		lbl := fmt.Sprintf("N%d", i)
		metrics := fnt.Metrics()
		textHeight := float32(metrics.HAscent + metrics.HDescent)

		drawText(screen, lbl, fnt, int(nAbsX)+9, int(ny-textHeight/2), ColorNeuronLabel)
		e.nodeHits = append(e.nodeHits, geNodeHit{cx: nAbsX, cy: ny, typ: simulation.NEURON, id: byte(i)})
	}

	// Action nodes
	for i := 0; i < numA; i++ {
		ay := aY[i]
		vector.FillCircle(screen, aAbsX, ay, geNodeR, ColorActionNode, false)

		lbl := nnActionName(byte(i))
		metrics := fnt.Metrics()
		textHeight := float32(metrics.HAscent + metrics.HDescent)

		drawText(screen, lbl, fnt, int(aAbsX)+9, int(ay-textHeight/2), ColorActionLabel)
		e.nodeHits = append(e.nodeHits, geNodeHit{cx: aAbsX, cy: ay, typ: simulation.ACTION, id: byte(i)})
	}

	// ── Controls strip ────────────────────────────────────────────────────────
	vector.FillRect(screen, nnX, ctrlY, geNNW, geNNControlH, ColorCtrlStripBG, false)
	vector.StrokeRect(screen, nnX, ctrlY, geNNW, geNNControlH, 1, ColorCtrlStripBorder, false)

	// +N / -N buttons
	bnx := nnX + gePad
	bny := ctrlY + gePad
	e.addNeuronBtn.Draw(screen, bnx, bny)
	e.remNeuronBtn.Draw(screen, bnx+36, bny)

	drawText(screen, fmt.Sprintf("Neurons: %d/%d", e.genome.CognitiveBreadth, e.params.Neurology.MaxSynapticDensity),
		fnt, int(bnx)+74, int(bny)+16, ColorWeightLabel)

	// Delete-edge button (only visible when edge is selected)
	if e.selEdgeIdx >= 0 {
		dbx := nnX + geNNW - 82
		e.delEdgeBtn.Draw(screen, dbx, bny)
	}

	// Second row of controls: weight slider or status hint
	row2Y := ctrlY + gePad + 28

	if e.selEdgeIdx >= 0 && e.selEdgeIdx < len(e.genome.Brain) {
		gene := e.genome.Brain[e.selEdgeIdx]
		w := gene.WeightAsFloat32() // [-1, +1]

		labelEndX := nnX + gePad + 54
		drawText(screen, "Weight:", fnt, int(nnX)+gePad, int(row2Y)+13, ColorWeightLabel)

		trkW := float32(geNNW) - (labelEndX - nnX) - 54
		e.wtTrackX = labelEndX
		e.wtTrackY = row2Y + 4
		e.wtTrackW = trkW

		// Track
		vector.FillRect(screen, e.wtTrackX, e.wtTrackY, e.wtTrackW, 6, ColorTraitTrackBG, false)
		// Fill (0 = left=-1, 1 = right=+1)
		tFrac := (w + 1.0) / 2.0
		fillClr := ColorWeightPos
		if w < 0 {
			fillClr = ColorWeightNeg
		}
		vector.FillRect(screen, e.wtTrackX, e.wtTrackY, e.wtTrackW*tFrac, 6, fillClr, false)
		// Knob
		knobX := e.wtTrackX + e.wtTrackW*tFrac - 3
		vector.FillRect(screen, knobX, e.wtTrackY-2, 6, 10, ColorWeightKnob, false)
		// Value label
		drawText(screen, fmt.Sprintf("%.2f", w), fnt,
			int(e.wtTrackX+e.wtTrackW)+6, int(row2Y)+13, ColorWeightLabel)
	} else if e.pendingSrc != nil {
		var srcDesc string
		switch e.pendingSrc.typ {
		case simulation.SENSOR:
			srcDesc = "Sensor: " + nnSensorName(e.pendingSrc.id)
		case simulation.NEURON:
			srcDesc = fmt.Sprintf("Neuron: N%d", e.pendingSrc.id)
		}
		drawText(screen, "Click target to connect from "+srcDesc,
			fnt, int(nnX)+gePad, int(row2Y)+13, ColorPendingHint)
	} else {
		drawText(screen, "Click node to start connection  |  Click edge dot to select",
			fnt, int(nnX)+gePad, int(row2Y)+13, ColorHintMuted)
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
