package colors

import "image/color"

var (
	// Menu bar
	ColorMenuBar = color.RGBA{12, 14, 28, 220}

	// Dropdown panels
	ColorDropdownBG     = color.RGBA{12, 14, 28, 235}
	ColorDropdownBorder = color.RGBA{90, 90, 150, 255}

	// Modal panels (genome editor, saved genomes)
	ColorModalBG       = color.RGBA{8, 10, 22, 248}
	ColorModalBorder   = color.RGBA{90, 90, 155, 255}
	ColorModalTitleBar = color.RGBA{18, 18, 48, 255}
	ColorModalTitle    = color.RGBA{200, 200, 255, 255}

	// Detail / genome panel
	ColorDetailPanelBG     = color.RGBA{8, 10, 22, 215}
	ColorDetailPanelBorder = color.RGBA{90, 90, 150, 255}

	// Stats panel
	ColorStatPanelBG     = color.RGBA{8, 10, 22, 160}
	ColorStatPanelBorder = color.RGBA{50, 60, 90, 180}

	// Food types (used across dropdowns and creature detail)
	ColorFoliage = color.RGBA{65, 180, 55, 255}
	ColorFungi   = color.RGBA{160, 80, 200, 255}
	ColorMeat    = color.RGBA{215, 60, 60, 255}

	// Energy / health bars
	ColorEnergyHigh = color.RGBA{55, 185, 55, 255}
	ColorEnergyLow  = color.RGBA{190, 55, 55, 255}

	// Specialised stat bars
	ColorDopamineHigh = color.RGBA{216, 27, 96, 255}
	ColorDopamineLow  = color.RGBA{48, 63, 159, 255}
	ColorResponseHigh = color.RGBA{255, 180, 40, 255}
	ColorResponseLow  = color.RGBA{60, 60, 200, 255}

	// Text / labels
	ColorLabelPrimary   = color.RGBA{255, 220, 80, 255}
	ColorLabelSecondary = color.RGBA{175, 175, 215, 255}
	ColorLabelMuted     = color.RGBA{120, 120, 180, 255}
	ColorLabelInfo      = color.RGBA{180, 220, 255, 255}
	ColorLabelSubtle    = color.RGBA{155, 175, 195, 255}
	ColorLabelTargetE   = color.RGBA{255, 230, 50, 255}
	ColorLabelGreen     = color.RGBA{80, 210, 100, 255}
	ColorLabelMeatRed   = color.RGBA{210, 90, 90, 255}
	ColorInfoBlue       = color.RGBA{100, 180, 255, 255}

	// Reproduction labels
	ColorReproAsexual = color.RGBA{100, 180, 255, 255}
	ColorReproSexual  = color.RGBA{255, 120, 180, 255}

	// Modal buttons
	ColorBtnClose     = color.RGBA{160, 50, 50, 200}
	ColorBtnCancel    = color.RGBA{80, 40, 40, 220}
	ColorBtnSave      = color.RGBA{40, 100, 60, 220}
	ColorBtnEdit      = color.RGBA{40, 60, 130, 220}
	ColorBtnAddNeuron = color.RGBA{40, 100, 60, 220}
	ColorBtnRemNeuron = color.RGBA{100, 40, 40, 220}
	ColorBtnDelEdge   = color.RGBA{140, 40, 40, 220}
	ColorBtnAsexual   = color.RGBA{60, 80, 160, 200}
	ColorBtnSexual    = color.RGBA{160, 70, 70, 200}

	// Genome editor structural elements
	ColorSeparator       = color.RGBA{45, 45, 80, 255}
	ColorFooterBG        = color.RGBA{14, 14, 34, 255}
	ColorCtrlStripBG     = color.RGBA{12, 12, 30, 210}
	ColorCtrlStripBorder = color.RGBA{38, 38, 68, 220}

	// Trait sliders (genome editor left column)
	ColorTraitTrackBG   = color.RGBA{38, 38, 58, 255}
	ColorTraitTrackFill = color.RGBA{75, 135, 205, 255}
	ColorTraitKnob      = color.RGBA{185, 205, 240, 255}
	ColorTraitValue     = color.RGBA{155, 175, 195, 255}

	// Name / text input
	ColorInputBG            = color.RGBA{18, 18, 40, 255}
	ColorInputBorder        = color.RGBA{55, 55, 90, 255}
	ColorInputBorderFocused = color.RGBA{100, 110, 210, 255}
	ColorInputPlaceholder   = color.RGBA{90, 90, 120, 255}

	// Neural network nodes and edges
	ColorSensorNode  = color.RGBA{80, 150, 220, 255}
	ColorSensorLabel = color.RGBA{155, 175, 215, 200}
	ColorNeuronNode  = color.RGBA{200, 180, 80, 255}
	ColorNeuronLabel = color.RGBA{195, 175, 80, 200}
	ColorActionNode  = color.RGBA{220, 100, 80, 255}
	ColorActionLabel = color.RGBA{215, 155, 145, 200}
	ColorNodeSelect  = color.RGBA{255, 255, 80, 255}
	ColorSensorHdr   = color.RGBA{100, 140, 200, 200}
	ColorNeuronHdr   = color.RGBA{180, 160, 80, 200}
	ColorActionHdr   = color.RGBA{200, 100, 80, 200}
	ColorWeightPos   = color.RGBA{75, 200, 75, 255}
	ColorWeightNeg   = color.RGBA{200, 75, 75, 255}
	ColorWeightKnob  = color.RGBA{220, 220, 255, 255}
	ColorWeightLabel = color.RGBA{155, 175, 215, 220}
	ColorPendingLine = color.RGBA{255, 255, 80, 130}
	ColorPendingHint = color.RGBA{255, 255, 80, 210}
	ColorHintMuted   = color.RGBA{110, 110, 170, 200}

	// Saved genomes panel
	ColorSavedRowSep   = color.RGBA{35, 35, 60, 180}
	ColorSavedRowHover = color.RGBA{30, 30, 60, 120}
	ColorSavedName     = color.RGBA{200, 210, 255, 255}
	ColorSavedSummary  = color.RGBA{120, 130, 160, 200}
	ColorArrowEnabled  = color.RGBA{80, 80, 160, 220}
	ColorArrowDisabled = color.RGBA{60, 60, 100, 200}
	ColorScrollCount   = color.RGBA{100, 100, 150, 200}

	// Spawn crosshair cursor
	ColorCrosshair       = color.RGBA{255, 220, 80, 220}
	ColorCrosshairCircle = color.RGBA{255, 220, 80, 160}

	// Feedback toast
	ColorSaveFeedback = color.RGBA{100, 255, 120, 255}

	// Climate dropdown
	ColorClimateCool    = color.RGBA{120, 200, 255, 255}
	ColorClimateHot     = color.RGBA{255, 120, 60, 255}
	ColorClimateColdMp  = color.RGBA{100, 180, 255, 255}
	ColorClimateWarmBMR = color.RGBA{255, 160, 60, 255}

	// Spawn dropdown
	ColorSpawnTitle = color.RGBA{180, 100, 255, 255}

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
