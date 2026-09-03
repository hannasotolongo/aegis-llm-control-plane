package predictor

type Uncertainty struct {
	MemoryError  float64
	ComputeError float64
	SampleCount  int
}

type ErrorSamples struct {
	ActualMemory     []float64
	PredictedMemory  []float64
	ActualCompute    []float64
	PredictedCompute []float64
}

type UncertaintyEstimator interface {
	Estimate(
		samples ErrorSamples,
	) (Uncertainty, error)
}
