package simulation

const (
	MOVE byte = iota // positive = forward, negative = backward
	ROTATE           // positive = left (CCW), negative = right (CW)
	SET_OSCILLATOR_PERIOD
	SET_RESPONSIVENESS
	SET_LEARNING_RATE // modulates per-tick learning rate relative to genome baseline
	REST
	ATTACK
	REWARD
	PUNISH
	REPRODUCE // signal to reproduce this tick (asexual: splits daughter cell; sexual: seeks mate)

	ACTION_COUNT
)

func IsActionEnabled(a byte) bool {
	return a < ACTION_COUNT
}
