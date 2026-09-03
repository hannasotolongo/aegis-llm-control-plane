package predictor

import (
	"errors"
	"testing"
)

func TestMeanAbsoluteErrorEstimatorCalculatesSeparateErrors(t *testing.T) {
	estimator := NewMeanAbsoluteErrorEstimator()

	uncertainty, err := estimator.Estimate(
		ErrorSamples{
			ActualMemory: []float64{
				70,
				80,
				90,
			},
			PredictedMemory: []float64{
				65,
				84,
				82,
			},
			ActualCompute: []float64{
				60,
				70,
				80,
			},
			PredictedCompute: []float64{
				58,
				74,
				75,
			},
		},
	)
	if err != nil {
		t.Fatalf("estimate failed: %v", err)
	}

	expectedMemoryError := 17.0 / 3.0
	expectedComputeError := 11.0 / 3.0

	if uncertainty.MemoryError != expectedMemoryError {
		t.Fatalf(
			"expected memory error %.2f, got %.2f",
			expectedMemoryError,
			uncertainty.MemoryError,
		)
	}

	if uncertainty.ComputeError != expectedComputeError {
		t.Fatalf(
			"expected compute error %.2f, got %.2f",
			expectedComputeError,
			uncertainty.ComputeError,
		)
	}

	if uncertainty.SampleCount != 3 {
		t.Fatalf(
			"expected sample count 3, got %d",
			uncertainty.SampleCount,
		)
	}
}

func TestMeanAbsoluteErrorEstimatorRejectsEmptySamples(t *testing.T) {
	estimator := NewMeanAbsoluteErrorEstimator()

	_, err := estimator.Estimate(
		ErrorSamples{},
	)

	if !errors.Is(
		err,
		ErrInvalidErrorSamples,
	) {
		t.Fatalf(
			"expected ErrInvalidErrorSamples, got %v",
			err,
		)
	}
}

func TestMeanAbsoluteErrorEstimatorRejectsUnequalPairs(t *testing.T) {
	estimator := NewMeanAbsoluteErrorEstimator()

	_, err := estimator.Estimate(
		ErrorSamples{
			ActualMemory: []float64{
				70,
				80,
			},
			PredictedMemory: []float64{
				70,
			},
			ActualCompute: []float64{
				60,
				70,
			},
			PredictedCompute: []float64{
				60,
				70,
			},
		},
	)

	if !errors.Is(
		err,
		ErrInvalidErrorSamples,
	) {
		t.Fatalf(
			"expected ErrInvalidErrorSamples, got %v",
			err,
		)
	}
}

func TestMeanAbsoluteErrorEstimatorRejectsDifferentSeriesLengths(t *testing.T) {
	estimator := NewMeanAbsoluteErrorEstimator()

	_, err := estimator.Estimate(
		ErrorSamples{
			ActualMemory: []float64{
				70,
				80,
				90,
			},
			PredictedMemory: []float64{
				70,
				80,
				90,
			},
			ActualCompute: []float64{
				60,
				70,
			},
			PredictedCompute: []float64{
				60,
				70,
			},
		},
	)

	if !errors.Is(
		err,
		ErrInvalidErrorSamples,
	) {
		t.Fatalf(
			"expected ErrInvalidErrorSamples, got %v",
			err,
		)
	}
}
