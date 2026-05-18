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
	sgPanH    = 500
	sgTitleH  = 28
	sgPad     = 8
	sgRowH    = 34
	sgMaxRows = 12 // visible rows without scrolling
	sgBtnW    = 70
)

// SavedGenomesPanel is a modal list of persisted genomes the user can spawn.
type SavedGenomesPanel struct {
	visible bool
	genomes []simulation.NamedGenome
	scroll  int // index of the first visible row
	onSpawn func(*simulation.Genome, float32)

	// Geometry written during Draw, read during HandleInput
	panX, panY float32
	closeBtn   [4]float32
	rowBounds  [][4]float32 // bounds of visible spawn-button hit zones (one per visible row)
	upBtn      [4]float32
	downBtn    [4]float32
}

func newSavedGenomesPanel(onSpawn func(*simulation.Genome, float32)) *SavedGenomesPanel {
	return &SavedGenomesPanel{onSpawn: onSpawn}
}

// Open (re)loads the genome list from disk and shows the panel.
func (p *SavedGenomesPanel) Open() {
	genomes, _ := simulation.LoadAllCreatureGenomesNamed()
	p.genomes = genomes
	p.scroll = 0
	p.visible = true
}

// Scroll moves the list by delta rows (negative = up, positive = down).
func (p *SavedGenomesPanel) Scroll(delta int) {
	if !p.visible {
		return
	}
	p.scroll -= delta // wheel up → negative delta → show earlier rows
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
			if idx < len(p.genomes) && p.onSpawn != nil {
				p.onSpawn(p.genomes[idx].Genome, p.genomes[idx].Generation)
			}
			return true
		}
	}
	// Block clicks that land inside the panel
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

	// Derive button and text heights from font metrics.
	m := fnt.Metrics()
	glyphH := float32(m.HAscent + m.HDescent)
	textH := float32(m.HLineGap + m.HAscent + m.HDescent)
	btnH := glyphH + 6 // button height sized to the glyph

	// vertCenter returns the top offset that centres text of height textH inside boxH.
	vertCenter := func(boxH float32) int { return int((boxH - textH) / 2) }

	// Background + border
	vector.FillRect(screen, px, py, sgPanW, sgPanH, ColorModalBG, false)
	vector.StrokeRect(screen, px, py, sgPanW, sgPanH, 2, ColorModalBorder, false)

	// Title bar
	vector.FillRect(screen, px, py, sgPanW, sgTitleH, ColorModalTitleBar, false)
	drawText(screen, "SAVED GENOMES", fnt, int(px)+sgPad, int(py)+vertCenter(sgTitleH), ColorModalTitle)

	// Close [×] — width fixed at 24px (design choice), height from font
	const closeBtnW = float32(24)
	cbx := px + sgPanW - closeBtnW - 4
	cby := py + 3
	p.closeBtn = [4]float32{cbx, cby, closeBtnW, btnH}
	vector.FillRect(screen, cbx, cby, closeBtnW, btnH, ColorBtnClose, false)
	drawText(screen, "×", fnt, int(cbx)+6, int(cby)+vertCenter(btnH), color.White)

	// Scroll arrows — square buttons sized to btnH
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

	// Row area
	listY := py + sgTitleH + sgPad
	listH := arrowY - sgPad - listY

	// Empty state
	if len(p.genomes) == 0 {
		drawText(screen, "No saved genomes found in data/creatures/",
			fnt, int(px)+sgPad, int(listY)+sgPad, ColorSavedSummary)
		p.rowBounds = p.rowBounds[:0]
		return
	}

	// Scroll info
	total := len(p.genomes)
	end := p.scroll + sgMaxRows
	if end > total {
		end = total
	}

	cmx, cmy := ebiten.CursorPosition()
	cfx, cfy := float32(cmx), float32(cmy)

	// Vertical offsets for two lines of text within a row.
	twoLineH := 2*glyphH + 2
	rowLine1Off := int((float32(sgRowH) - twoLineH) / 2)
	rowLine2Off := rowLine1Off + int(glyphH) + 2

	p.rowBounds = p.rowBounds[:0]
	for i := p.scroll; i < end; i++ {
		ng := p.genomes[i]
		rowY := listY + float32(i-p.scroll)*sgRowH

		if rowY+sgRowH > listY+listH {
			break
		}

		// Row background (hover highlight)
		spawnBtnX := px + sgPanW - sgPad - sgBtnW
		rowBg := color.RGBA{0, 0, 0, 0}
		if cfx >= px+sgPad && cfx <= spawnBtnX-4 && cfy >= rowY && cfy <= rowY+sgRowH-2 {
			rowBg = ColorSavedRowHover
		}
		if rowBg.A > 0 {
			vector.FillRect(screen, px+sgPad, rowY, spawnBtnX-px-sgPad*2, sgRowH-2, rowBg, false)
		}

		// Separator
		vector.FillRect(screen, px+sgPad, rowY+sgRowH-1, float32(sgPanW)-sgPad*2, 1, ColorSavedRowSep, false)

		// Name + trait summary, vertically centred as a two-line block
		drawText(screen, ng.Name, fnt, int(px)+sgPad+4, int(rowY)+rowLine1Off, ColorSavedName)
		summary := fmt.Sprintf("Mass %d  Neurons %d  Genes %d  Gen %.1f",
			ng.Genome.BodyMass, ng.Genome.CognitiveBreadth, len(ng.Genome.Brain), ng.Generation)
		drawText(screen, summary, fnt, int(px)+sgPad+4, int(rowY)+rowLine2Off, ColorSavedSummary)

		// Spawn button — height from font metrics
		sbx := spawnBtnX
		sby := rowY + (sgRowH-btnH)/2
		p.rowBounds = append(p.rowBounds, [4]float32{sbx, sby, sgBtnW, btnH})
		vector.FillRect(screen, sbx, sby, sgBtnW, btnH, ColorBtnSave, false)
		drawText(screen, "Spawn", fnt, int(sbx)+8, int(sby)+vertCenter(btnH), color.White)
	}

	// Scroll counter, aligned with the arrow buttons
	drawText(screen, fmt.Sprintf("%d / %d", p.scroll+1, total), fnt,
		int(px)+sgPad, int(arrowY)+vertCenter(btnH), ColorScrollCount)
}
