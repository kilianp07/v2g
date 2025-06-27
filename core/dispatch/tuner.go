package dispatch

// LearningTuner adjusts SmartDispatcher parameters based on past dispatch results.
// Tune returns a new dispatcher with updated weights.
type LearningTuner interface {
	Tune(base SmartDispatcher, history []DispatchResult) SmartDispatcher
}

// NoopTuner returns the dispatcher unchanged.
type NoopTuner struct{}

func (NoopTuner) Tune(base SmartDispatcher, _ []DispatchResult) SmartDispatcher {
	return base
}
