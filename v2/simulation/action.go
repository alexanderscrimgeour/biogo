package simulation

const (
	// Positive = accelerate forward, negative = decelerate/reverse.
	ACCELERATE byte = iota
	// Positive = turn left (CCW), negative = turn right (CW).
	ROTATE
	// Triggers reproduction this tick: asexual splits a daughter cell; sexual seeks a mate.
	REPRODUCE
	// Adjusts the period of the OSC1 oscillator each tick.
	SET_OSCILLATOR_PERIOD
	// Scales the creature's responsiveness, amplifying or dampening all sensor signals.
	SET_RESPONSIVENESS
	// Modulates per-tick learning rate relative to the genome baseline.
	SET_LEARNING_RATE
	// Reduces energy expenditure; creature conserves energy while resting.
	REST
	// Attacks the nearest lighter creature ahead; transfers mass on a successful hit.
	ATTACK
	// Sends a reward signal to the nearest touching creature ahead.
	REWARD
	// Sends a punishment signal to the nearest touching creature ahead.
	PUNISH
	// Donates stomach contents proportional to action level to the nearest touching creature ahead.
	FEED

	ACTION_COUNT
)

func IsActionEnabled(a byte) bool {
	return a < ACTION_COUNT
}
