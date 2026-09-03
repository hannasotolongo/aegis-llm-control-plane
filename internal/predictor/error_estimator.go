package predictor

import "errors"

var ErrInvalidErrorSamples = errors.New(
	"actual and predicted samples must be non-empty and equal in length",
)

type MeanAbsoluteErrorEstimator struct{}

var _ UncertaintyEstimator = (*MeanAbsoluteErrorEstimator)(nil)

func NewMeanAbsoluteErrorEstimator() *MeanAbsoluteErrorEstimator {
	return &MeanAbsoluteErrorEstimator{}
}

func (e *MeanAbsoluteErrorEstimator) Estimate(
	samples ErrorSamples,
) (Uncertainty, error) {
	if !validSamplePair(
		samples.ActualMemory,
		samples.PredictedMemory,
	) ||
		!validSamplePair(
			samples.ActualCompute,
			samples.PredictedCompute,
		) ||
		len(samples.ActualMemory) != len(samples.ActualCompute) {
		return Uncertainty{}, ErrInvalidErrorSamples
	}

	memoryError := meanAbsoluteError(
		samples.ActualMemory,
		samples.PredictedMemory,
	)

	computeError := meanAbsoluteError(
		samples.ActualCompute,
		samples.PredictedCompute,
	)

	return Uncertainty{
		MemoryError:  memoryError,
		ComputeError: computeError,
		SampleCount:  len(samples.ActualMemory),
	}, nil
}

func validSamplePair(
	actual []float64,
	predicted []float64,
) bool {
	return len(actual) > 0 &&
		len(actual) == len(predicted)
}

func meanAbsoluteError(
	actual []float64,
	predicted []float64,
) float64 {
	var totalError float64

	for i := range actual {
		totalError += abs(
			actual[i] - predicted[i],
		)
	}

	return totalError / float64(len(actual))
}
