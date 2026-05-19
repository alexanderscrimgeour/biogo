package ui

import (
	"biogo/v2/simulation"
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	sgPanW    = 420
	sgPanH    = 640
	sgTitleH  = 28
	sgPad     = 8
	sgRowH    = 50 // tall enough for two lines of text with comfortable spacing
	sgMaxRows = 10
	sgBtnW    = 70
)

// SavedGenomesPanel is a modal list of persisted genomes the user can spawn or load.
type SavedGenomesPanel struct {
	visible  bool
	genomes  []simulation.NamedGenome
	scroll   int // index of the first visible row
	onSpawn  func(*simulation.Genome, float32)
	loadMode bool
	loadCb   func(*simulation.Genome, float32) // set when opened via OpenForLoad

	// Geometry written during Draw, read during HandleInput
	panX, panY float32
	closeBtn   [4]float32
	rowBounds  [][4]float32 // bounds of visible action-button hit zones
	upBtn      [4]float32
	downBtn    [4]float32
}

func newSavedGenomesPanel(onSpawn func(*simulation.Genome, float32)) *SavedGenomesPanel {
	return &SavedGenomesPanel{onSpawn: onSpawn}
}

// Open (re)loads the genome list from disk and shows the panel in spawn mode.
func (p *SavedGenomesPanel) Open() {
	genomes, _ := simulation.LoadAllCreatureGenomesNamed()
	p.genomes = genomes
	p.scroll = 0
	p.loadMode = false
	p.loadCb = nil
	p.visible = true
}

// OpenForLoad opens the panel with a custom callback instead of spawning.
func (p *SavedGenomesPanel) OpenForLoad(cb func(*simulation.Genome, float32)) {
	genomes, _ := simulation.LoadAllCreatureGenomesNamed()
	p.genomes = genomes
	p.scroll = 0
	p.loadMode = true
	p.loadCb = cb
	p.visible = true
}

// Scroll moves the list by delta rows (negative = up, positive = down).
func (p *SavedGenomesPanel) Scroll(delta int) {
	if !p.visible {
		return
	}
	p.scroll -= delta
	p.clampScroll()
}

func (p *SavedGenomesPanel) clampScroll() {
	max := len(p.genomes) - sgMaxRows
	if max < 0 {
		max = 0
	}
	if p.scroll > max {
		p.scroll = max
	}
	if p.scroll < 0 {
		p.scroll = 0
	}
}

// HandleInput processes a click. Returns true if consumed.
func (p *SavedGenomesPanel) HandleInput(mx, my int) bool {
	if !p.visible {
		return false
	}
	fx, fy := float32(mx), float32(my)

	if inGeRect(fx, fy, p.closeBtn) {
		p.visible = false
		return true
	}
	if inGeRect(fx, fy, p.upBtn) {
		p.scroll--
		p.clampScroll()
		return true
	}
	if inGeRect(fx, fy, p.downBtn) {
		p.scroll++
		p.clampScroll()
		return true
	}
	for i, b := range p.rowBounds {
		if inGeRect(fx, fy, b) {
			idx := p.scroll + i
			if idx < len(p.genomes) {
				ng := p.genomes[idx]
				if p.loadMode && p.loadCb != nil {
					p.loadCb(ng.Genome, ng.Generation)
				} else if p.onSpawn != nil {
					p.onSpawn(ng.Genome, ng.Generation)
				}
			}
			return true
		}
	}
	return fx >= p.panX && fx <= p.panX+sgPanW && fy >= p.panY && fy <= p.panY+sgPanH
}

// Draw renders the panel centred on screen.
func (p *SavedGenomesPanel) Draw(screen *ebiten.Image, fnt *textv2.GoXFace) {
	if !p.visible {
		return
	}

	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	p.panX = float32(sw/2 - sgPanW/2)
	p.panY = float32(sh/2 - sgPanH/2)
	px, py := p.panX, p.panY

	m := fnt.Metrics()
	glyphH := float32(m.HAscent + m.HDescent)
	btnH := glyphH + 8

	// vertCenter returns the y-offset that visually centres a single line of text within boxH.
	vertCenter := func(boxH float32) int { return int((boxH - glyphH) / 2) }

	vector.FillRect(screen, px, py, sgPanW, sgPanH, ColorModalBG, false)
	vector.StrokeRect(screen, px, py, sgPanW, sgPanH, 2, ColorModalBorder, false)

	// Title bar
	vector.FillRect(screen, px, py, sgPanW, sgTitleH, ColorModalTitleBar, false)
	title := "SAVED GENOMES"
	if p.loadMode {
		title = "SAVED GENOMES — select to load into editor"
	}
	drawText(screen, title, fnt, int(px)+sgPad, int(py)+vertCenter(sgTitleH), ColorModalTitle)

	const closeBtnW = float32(24)
	cbx := px + sgPanW - closeBtnW - 4
	cby := py + (sgTitleH-btnH)/2
	p.closeBtn = [4]float32{cbx, cby, closeBtnW, btnH}
	vector.FillRect(screen, cbx, cby, closeBtnW, btnH, ColorBtnClose, false)
	drawText(screen, "×", fnt, int(cbx)+6, int(cby)+vertCenter(btnH), color.White)

	// Scroll arrows
	arrowY := py + sgPanH - sgPad - btnH
	upBtnX := px + sgPanW/2 - btnH - 2
	p.upBtn = [4]float32{upBtnX, arrowY, btnH, btnH}
	p.downBtn = [4]float32{upBtnX + btnH + 4, arrowY, btnH, btnH}

	upClr := ColorArrowDisabled
	if p.scroll > 0 {
		upClr = ColorArrowEnabled
	}
	downClr := ColorArrowDisabled
	if p.scroll < len(p.genomes)-sgMaxRows {
		downClr = ColorArrowEnabled
	}
	vector.FillRect(screen, p.upBtn[0], p.upBtn[1], p.upBtn[2], p.upBtn[3], upClr, false)
	drawText(screen, "▲", fnt, int(upBtnX)+4, int(arrowY)+vertCenter(btnH), color.White)
	vector.FillRect(screen, p.downBtn[0], p.downBtn[1], p.downBtn[2], p.downBtn[3], downClr, false)
	drawText(screen, "▼", fnt, int(p.downBtn[0])+4, int(arrowY)+vertCenter(btnH), color.White)

	listY := py + sgTitleH + sgPad
	listH := arrowY - sgPad - listY

	if len(p.genomes) == 0 {
		drawText(screen, "No saved genomes found in data/creatures/",
			fnt, int(px)+sgPad, int(listY)+sgPad, ColorSavedSummary)
		p.rowBounds = p.rowBounds[:0]
		return
	}

	total := len(p.genomes)
	end := p.scroll + sgMaxRows
	if end > total {
		end = total
	}

	cmx, cmy := ebiten.CursorPosition()
	cfx, cfy := float32(cmx), float32(cmy)

	// Two text lines centred vertically within each row.
	// Line 1 top: rowTop + (sgRowH - 2*glyphH - gap) / 2
	const lineGap = float32(4)
	twoLinesH := 2*glyphH + lineGap
	line1Off := int((float32(sgRowH) - twoLinesH) / 2)
	line2Off := line1Off + int(glyphH) + int(lineGap)

	actionLabel := "Spawn"
	if p.loadMode {
		actionLabel = "Load"
	}

	p.rowBounds = p.rowBounds[:0]
	for i := p.scroll; i < end; i++ {
		ng := p.genomes[i]
		rowY := listY + float32(i-p.scroll)*sgRowH

		if rowY+sgRowH > listY+listH {
			break
		}

		spawnBtnX := px + sgPanW - sgPad - sgBtnW
		rowBg := color.RGBA{0, 0, 0, 0}
		if cfx >= px+sgPad && cfx <= spawnBtnX-4 && cfy >= rowY && cfy <= rowY+sgRowH-2 {
			rowBg = ColorSavedRowHover
		}
		if rowBg.A > 0 {
			vector.FillRect(screen, px+sgPad, rowY, spawnBtnX-px-sgPad*2, sgRowH-2, rowBg, false)
		}

		// Row separator
		vector.FillRect(screen, px+sgPad, rowY+sgRowH-1, float32(sgPanW)-sgPad*2, 1, ColorSavedRowSep, false)

		// Name + summary, two-line block centred in row
		drawText(screen, ng.Name, fnt, int(px)+sgPad+4, int(rowY)+line1Off, ColorSavedName)
		summary := fmt.Sprintf("Mass %d  Neurons %d  Genes %d  Gen %.1f",
			ng.Genome.BodyMass, ng.Genome.CognitiveBreadth, len(ng.Genome.Brain), ng.Generation)
		drawText(screen, summary, fnt, int(px)+sgPad+4, int(rowY)+line2Off, ColorSavedSummary)

		// Action button
		sbx := spawnBtnX
		sby := rowY + (sgRowH-btnH)/2
		p.rowBounds = append(p.rowBounds, [4]float32{sbx, sby, sgBtnW, btnH})
		vector.FillRect(screen, sbx, sby, sgBtnW, btnH, ColorBtnSave, false)
		drawText(screen, actionLabel, fnt, int(sbx)+8, int(sby)+vertCenter(btnH), color.White)
	}

	drawText(screen, fmt.Sprintf("%d / %d", p.scroll+1, total), fnt,
		int(px)+sgPad, int(arrowY)+vertCenter(btnH), ColorScrollCount)
}
