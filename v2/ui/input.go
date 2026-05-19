package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type InteractionController struct {
	camera     *Camera
	sim        SimulationState
	dragging   bool
	dragMoved  bool
	dragStartX int
	dragStartY int
	dragLastX  int
	dragLastY  int
}

func NewInteractionController(cam *Camera, sim SimulationState) *InteractionController {
	return &InteractionController{camera: cam, sim: sim}
}

// HandleContinuousInput runs every frame to process camera panning and tracking
func (ic *InteractionController) HandleContinuousInput(sliderDragging bool) {
	mx, my := ebiten.CursorPosition()

	if !sliderDragging && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && ic.dragging {
		dx := mx - ic.dragLastX
		dy := my - ic.dragLastY

		// 3x3 pixel threshold check
		if !ic.dragMoved && (dx*dx+dy*dy > 9) {
			ic.dragMoved = true
		}

		if ic.dragMoved {
			ic.camera.X -= float64(dx) / ic.camera.Zoom
			ic.camera.Y -= float64(dy) / ic.camera.Zoom
		}
		ic.dragLastX, ic.dragLastY = mx, my
	}

	const panSpeed = 5.0
	pan := panSpeed / ic.camera.Zoom
	if ebiten.IsKeyPressed(ebiten.KeyW) {
		ic.camera.Y -= pan
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		ic.camera.Y += pan
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		ic.camera.X -= pan
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		ic.camera.X += pan
	}

	worldW := ic.sim.WorldWidth() * float64(UnitSize)
	worldH := ic.sim.WorldHeight() * float64(UnitSize)
	ic.camera.Clamp(worldW, worldH)
}

func (ic *InteractionController) Update(sliderDragging bool) {
	mx, my := ebiten.CursorPosition()

	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		if !sliderDragging {
			if !ic.dragging {

				ic.StartCameraDrag(mx, my)
			} else {

				dx := mx - ic.dragLastX
				dy := my - ic.dragLastY

				if !ic.dragMoved && (dx*dx+dy*dy > 9) {
					ic.dragMoved = true
				}

				if ic.dragMoved {
					ic.camera.X -= float64(dx) / ic.camera.Zoom
					ic.camera.Y -= float64(dy) / ic.camera.Zoom
				}
				ic.dragLastX, ic.dragLastY = mx, my
			}
		}
	} else {
		if ic.dragging {

			ic.StopCameraDrag()
		}
	}

	const panSpeed = 5.0
	pan := panSpeed / ic.camera.Zoom
	if ebiten.IsKeyPressed(ebiten.KeyW) {
		ic.camera.Y -= pan
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		ic.camera.Y += pan
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		ic.camera.X -= pan
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		ic.camera.X += pan
	}

	worldW := ic.sim.WorldWidth() * float64(UnitSize)
	worldH := ic.sim.WorldHeight() * float64(UnitSize)
	ic.camera.Clamp(worldW, worldH)
}

func (ic *InteractionController) StartCameraDrag(x, y int) {
	ic.dragging = true
	ic.dragStartX, ic.dragStartY = x, y
	ic.dragLastX, ic.dragLastY = x, y
	ic.dragMoved = false
}

func (ic *InteractionController) StopCameraDrag() bool {
	moved := ic.dragMoved
	ic.dragging = false
	return moved
}

func (ic *InteractionController) CamDragStartPos() (int, int) {
	return ic.dragStartX, ic.dragStartY
}
