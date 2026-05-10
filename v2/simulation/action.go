package simulation

const (
	MOVE byte = iota // positive = forward, negative = backward
	ROTATE           // positive = left (CCW), negative = right (CW)
	SET_OSCILLATOR_PERIOD
	SET_RESPONSIVENESS
	SET_LEARNING_RATE // modulates per-tick learning rate relative to genome baseline
	DO_NOTHING

	ACTION_COUNT
)

func IsActionEnabled(a byte) bool {
	return a < ACTION_COUNT
}
