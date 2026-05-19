package components

import "image/color"

var (
	// Button states
	ColorDefault       = color.RGBA{120, 144, 156, 100}
	ColorButtonPressed = color.RGBA{144, 164, 174, 100}
	ColorButtonGreen   = color.RGBA{76, 175, 80, 100}
	ColorButtonRed     = color.RGBA{244, 67, 54, 100}

	// Slider
	ColorSliderBG  = color.RGBA{30, 30, 50, 220}
	ColorTrackBG   = color.RGBA{60, 60, 80, 255}
	ColorTrackFill = color.RGBA{80, 140, 210, 255}

	// Bars (energy, proportion)
	ColorBarBG = color.RGBA{35, 35, 35, 255}

	// GenomeBar
	ColorGenomeBarBG      = color.RGBA{20, 20, 45, 200}
	ColorGenomeBarFill    = color.RGBA{55, 160, 210, 220}
	ColorGenomeBinAsexual = color.RGBA{200, 80, 30, 180}
	ColorGenomeBinSexual  = color.RGBA{60, 110, 210, 180}
	ColorGenomeBinDim     = color.RGBA{25, 25, 50, 160}
)
