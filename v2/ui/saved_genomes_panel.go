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
	onSpawn func(*simulation.Genome)

	// Geometry written during Draw, read during HandleInput
	panX, panY float32
	closeBtn   [4]float32
	rowBounds  [][4]float32 // bounds of visible spawn-button hit zones (one per visible row)
	upBtn      [4]float32
	downBtn    [4]float32
}

func newSavedGenomesPanel(onSpawn func(*simulation.Genome)) *SavedGenomesPanel {
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
				p.onSpawn(p.genomes[idx].Genome)
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

	// Background + border
	vector.FillRect(screen, px, py, sgPanW, sgPanH, color.RGBA{8, 10, 22, 248}, false)
	vector.StrokeRect(screen, px, py, sgPanW, sgPanH, 2, color.RGBA{90, 90, 155, 255}, false)

	// Title bar
	vector.FillRect(screen, px, py, sgPanW, sgTitleH, color.RGBA{18, 18, 48, 255}, false)
	drawText(screen, "SAVED GENOMES", fnt, int(px)+sgPad, int(py)+18, color.RGBA{200, 200, 255, 255})

	// Close [×]
	cbx := px + sgPanW - 28
	cby := py + 3
	p.closeBtn = [4]float32{cbx, cby, 24, 22}
	vector.FillRect(screen, cbx, cby, 24, 22, color.RGBA{160, 50, 50, 200}, false)
	drawText(screen, "×", fnt, int(cbx)+6, int(cby)+16, color.White)

	// Scroll arrows (always present)
	arrowY := py + sgPanH - sgPad - 22
	upBtnX := px + sgPanW/2 - 26
	p.upBtn = [4]float32{upBtnX, arrowY, 24, 22}
	p.downBtn = [4]float32{upBtnX + 28, arrowY, 24, 22}

	upClr := color.RGBA{60, 60, 100, 200}
	if p.scroll > 0 {
		upClr = color.RGBA{80, 80, 160, 220}
	}
	downClr := color.RGBA{60, 60, 100, 200}
	if p.scroll < len(p.genomes)-sgMaxRows {
		downClr = color.RGBA{80, 80, 160, 220}
	}
	vector.FillRect(screen, p.upBtn[0], p.upBtn[1], p.upBtn[2], p.upBtn[3], upClr, false)
	drawText(screen, "▲", fnt, int(upBtnX)+4, int(arrowY)+16, color.White)
	vector.FillRect(screen, p.downBtn[0], p.downBtn[1], p.downBtn[2], p.downBtn[3], downClr, false)
	drawText(screen, "▼", fnt, int(upBtnX)+32, int(arrowY)+16, color.White)

	// Row area
	listY := py + sgTitleH + sgPad
	listH := arrowY - sgPad - listY

	// Empty state
	if len(p.genomes) == 0 {
		drawText(screen, "No saved genomes found in data/creatures/",
			fnt, int(px)+sgPad, int(listY)+20, color.RGBA{120, 120, 170, 200})
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
			rowBg = color.RGBA{30, 30, 60, 120}
		}
		if rowBg.A > 0 {
			vector.FillRect(screen, px+sgPad, rowY, spawnBtnX-px-sgPad*2, sgRowH-2, rowBg, false)
		}

		// Separator
		vector.FillRect(screen, px+sgPad, rowY+sgRowH-1, float32(sgPanW)-sgPad*2, 1, color.RGBA{35, 35, 60, 180}, false)

		// Name + trait summary
		nameClr := color.RGBA{200, 210, 255, 255}
		drawText(screen, ng.Name, fnt, int(px)+sgPad+4, int(rowY)+15, nameClr)
		summary := fmt.Sprintf("Mass %d  Neurons %d  Genes %d",
			ng.Genome.Mass, ng.Genome.CognitiveBreadth, len(ng.Genome.Brain))
		drawText(screen, summary, fnt, int(px)+sgPad+4, int(rowY)+28, color.RGBA{120, 130, 160, 200})

		// Spawn button
		sbx := spawnBtnX
		sby := rowY + (sgRowH-22)/2
		p.rowBounds = append(p.rowBounds, [4]float32{sbx, sby, sgBtnW, 22})
		vector.FillRect(screen, sbx, sby, sgBtnW, 22, color.RGBA{40, 100, 60, 220}, false)
		drawText(screen, "Spawn", fnt, int(sbx)+8, int(sby)+15, color.White)
	}

	// Scroll counter
	drawText(screen, fmt.Sprintf("%d / %d", p.scroll+1, total), fnt,
		int(px)+sgPad, int(arrowY)+16, color.RGBA{100, 100, 150, 200})
}
