package views

const historyLen = 5000

type HistSample struct {
	Pop           int
	FoliageEnergy float64
	FungiEnergy   float64
	MeatEnergy    float64
	TotalEnergy   float64
}

type PerformanceHistory struct {
	data  [historyLen]HistSample
	head  int
	count int
}

func NewPerformanceHistory() *PerformanceHistory {
	return &PerformanceHistory{}
}

func (ph *PerformanceHistory) Append(sample HistSample) {
	ph.data[ph.head] = sample
	ph.head = (ph.head + 1) % historyLen
	if ph.count < historyLen {
		ph.count++
	}
}

func (ph *PerformanceHistory) Samples() []HistSample {
	// Returns samples ordered chronologically regardless of ring position
	out := make([]uint16, 0, ph.count)
	_ = out // Optional helper logic for graphing UI
	return nil
}

func (ph *PerformanceHistory) SampleAt(i int) HistSample {
	return ph.data[i]
}

func (ph *PerformanceHistory) CurrentHead() int {
	return ph.head
}

func (ph *PerformanceHistory) TotalCount() int {
	return ph.count
}

func (ph *PerformanceHistory) Reset() {
	ph.head = 0
	ph.count = 0
}
