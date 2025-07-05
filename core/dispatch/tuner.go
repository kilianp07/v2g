package dispatch

// LearningTuner adjusts SmartDispatcher parameters based on past dispatch results.
// history contains results of past dispatches.
type LearningTuner interface {
	Tune(history []DispatchResult)
}

// NoopTuner returns the dispatcher unchanged.
type NoopTuner struct{}

func (NoopTuner) Tune(_ []DispatchResult) {
}
