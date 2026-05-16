package simulation

const (
	// -- Tier 0 --

	// Positive = accelerate forward, negative = decelerate/reverse.
	ACCELERATE byte = iota
	// Positive = turn left (CCW), negative = turn right (CW).
	ROTATE

	// -- Tier 1 --

	// Adjusts the period of the OSC1 oscillator each tick.
	SET_OSCILLATOR_PERIOD
	// Reduces energy expenditure; creature conserves energy while resting.
	REST
	// Triggers reproduction this tick: asexual splits a daughter cell; sexual seeks a mate.
	// NOTE: Not necessary for survival until tier 3, however this is intended to positively reinforce
	// before they reach tier 3
	REPRODUCE

	// -- Tier 2 --

	// Attacks the nearest lighter creature ahead; transfers mass on a successful hit.
	ATTACK

	// -- Tier 3 --

	// Donates stomach contents proportional to action level to the nearest touching creature ahead.
	FEED
	// Scales the creature's responsiveness, amplifying or dampening all sensor signals.
	SET_RESPONSIVENESS
	// Modulates per-tick learning rate relative to the genome baseline.
	SET_LEARNING_RATE
	// Sends a reward signal to the nearest touching creature ahead.
	REWARD
	// Sends a punishment signal to the nearest touching creature ahead.
	PUNISH

	ACTION_COUNT
)

const (
	MaxTier0Action = 1
	MaxTier1Action = 4
	MaxTier2Action = 5
)

func IsActionEnabled(a byte) bool {
	return a < ACTION_COUNT
}
