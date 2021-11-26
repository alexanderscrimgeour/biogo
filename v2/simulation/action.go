package simulation

const (
	MOVE_X byte = iota
	MOVE_Y
	MOVE_FWD
	MOVE_RL
	MOVE_LEFT
	MOVE_RIGHT
	MOVE_RANDOM
	SET_OSCILLATOR_PERIOD
	SET_RESPONSIVENESS
	MOVE_EAST
	MOVE_WEST
	MOVE_NORTH
	MOVE_SOUTH

	ACTION_COUNT
	// Disabled for now
	REPRODUCE
	EAT
)

func IsActionEnabled(a byte) bool {
	return a < ACTION_COUNT
}
