package predictor

import (
	"errors"
	"testing"
	"time"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/telemetry"
)

func TestForecastRisingMemoryContention(t *testing.T) {
	predictor := NewTrendPredictor()

	baseTime := time.Date(
		2026,
		time.September,
		3,
		10,
		0,
		0,
		0,
		time.UTC,
	)

	records := []telemetry.Record{
		{
			Timestamp:          baseTime,
			WorkerID:           "worker-1",
			MemoryUtilization:  70,
			ComputeUtilization: 40,
		},
		{
			Timestamp:          baseTime.Add(1 * time.Second),
			WorkerID:           "worker-1",
			MemoryUtilization:  78,
			ComputeUtilization: 45,
		},
		{
			Timestamp:          baseTime.Add(2 * time.Second),
			WorkerID:           "worker-1",
			MemoryUtilization:  84,
			ComputeUtilization: 50,
		},
	}

	forecast, err := predictor.Forecast(
		records,
		1*time.Second,
	)
	if err != nil {
		t.Fatalf(
			"Forecast returned error: %v",
			err,
		)
	}

	if !forecast.PredictedContention {
		t.Fatal(
			"expected predicted contention",
		)
	}

	if forecast.PredictedMemoryUtilization <= 85 {
		t.Fatalf(
			"expected predicted memory above 85, got %.2f",
			forecast.PredictedMemoryUtilization,
		)
	}

	if forecast.WorkerID != "worker-1" {
		t.Fatalf(
			"expected worker-1, got %q",
			forecast.WorkerID,
		)
	}
}

func TestForecastStableWorker(t *testing.T) {
	predictor := NewTrendPredictor()

	baseTime := time.Date(
		2026,
		time.September,
		3,
		10,
		0,
		0,
		0,
		time.UTC,
	)

	records := []telemetry.Record{
		{
			Timestamp:          baseTime,
			WorkerID:           "worker-2",
			MemoryUtilization:  40,
			ComputeUtilization: 50,
		},
		{
			Timestamp:          baseTime.Add(1 * time.Second),
			WorkerID:           "worker-2",
			MemoryUtilization:  41,
			ComputeUtilization: 51,
		},
		{
			Timestamp:          baseTime.Add(2 * time.Second),
			WorkerID:           "worker-2",
			MemoryUtilization:  42,
			ComputeUtilization: 52,
		},
	}

	forecast, err := predictor.Forecast(
		records,
		1*time.Second,
	)
	if err != nil {
		t.Fatalf(
			"Forecast returned error: %v",
			err,
		)
	}

	if forecast.PredictedContention {
		t.Fatal(
			"did not expect predicted contention",
		)
	}

	if forecast.PredictedMemoryUtilization != 43 {
		t.Fatalf(
			"expected predicted memory 43, got %.2f",
			forecast.PredictedMemoryUtilization,
		)
	}

	if forecast.PredictedComputeUtilization != 53 {
		t.Fatalf(
			"expected predicted compute 53, got %.2f",
			forecast.PredictedComputeUtilization,
		)
	}
}

func TestForecastInsufficientHistory(t *testing.T) {
	predictor := NewTrendPredictor()

	records := []telemetry.Record{
		{
			WorkerID:          "worker-1",
			MemoryUtilization: 70,
		},
		{
			WorkerID:          "worker-1",
			MemoryUtilization: 80,
		},
	}

	_, err := predictor.Forecast(
		records,
		1*time.Second,
	)

	if !errors.Is(
		err,
		ErrInsufficientHistory,
	) {
		t.Fatalf(
			"expected ErrInsufficientHistory, got %v",
			err,
		)
	}
}

func TestForecastIncludesBacktestedUncertainty(t *testing.T) {
	predictor := NewTrendPredictor()

	baseTime := time.Date(
		2026,
		time.September,
		3,
		10,
		0,
		0,
		0,
		time.UTC,
	)

	records := []telemetry.Record{
		{
			Timestamp:          baseTime,
			WorkerID:           "worker-1",
			MemoryUtilization:  40,
			ComputeUtilization: 30,
		},
		{
			Timestamp:          baseTime.Add(1 * time.Second),
			WorkerID:           "worker-1",
			MemoryUtilization:  50,
			ComputeUtilization: 40,
		},
		{
			Timestamp:          baseTime.Add(2 * time.Second),
			WorkerID:           "worker-1",
			MemoryUtilization:  60,
			ComputeUtilization: 50,
		},
		{
			Timestamp:          baseTime.Add(3 * time.Second),
			WorkerID:           "worker-1",
			MemoryUtilization:  72,
			ComputeUtilization: 63,
		},
		{
			Timestamp:          baseTime.Add(4 * time.Second),
			WorkerID:           "worker-1",
			MemoryUtilization:  85,
			ComputeUtilization: 75,
		},
	}

	forecast, err := predictor.Forecast(
		records,
		1*time.Second,
	)
	if err != nil {
		t.Fatalf(
			"Forecast returned error: %v",
			err,
		)
	}

	if forecast.Uncertainty.SampleCount != 2 {
		t.Fatalf(
			"expected 2 uncertainty samples, got %d",
			forecast.Uncertainty.SampleCount,
		)
	}

	if forecast.Uncertainty.MemoryError <= 0 {
		t.Fatalf(
			"expected positive memory error, got %.2f",
			forecast.Uncertainty.MemoryError,
		)
	}

	if forecast.Uncertainty.ComputeError <= 0 {
		t.Fatalf(
			"expected positive compute error, got %.2f",
			forecast.Uncertainty.ComputeError,
		)
	}
}

func TestForecastWithMinimumHistoryHasNoUncertaintySamples(t *testing.T) {
	predictor := NewTrendPredictor()

	baseTime := time.Date(
		2026,
		time.September,
		3,
		10,
		0,
		0,
		0,
		time.UTC,
	)

	records := []telemetry.Record{
		{
			Timestamp:          baseTime,
			WorkerID:           "worker-1",
			MemoryUtilization:  40,
			ComputeUtilization: 30,
		},
		{
			Timestamp:          baseTime.Add(1 * time.Second),
			WorkerID:           "worker-1",
			MemoryUtilization:  50,
			ComputeUtilization: 40,
		},
		{
			Timestamp:          baseTime.Add(2 * time.Second),
			WorkerID:           "worker-1",
			MemoryUtilization:  60,
			ComputeUtilization: 50,
		},
	}

	forecast, err := predictor.Forecast(
		records,
		1*time.Second,
	)
	if err != nil {
		t.Fatalf(
			"Forecast returned error: %v",
			err,
		)
	}

	if forecast.Uncertainty.SampleCount != 0 {
		t.Fatalf(
			"expected 0 uncertainty samples, got %d",
			forecast.Uncertainty.SampleCount,
		)
	}

	if forecast.Uncertainty.MemoryError != 0 {
		t.Fatalf(
			"expected zero memory error, got %.2f",
			forecast.Uncertainty.MemoryError,
		)
	}

	if forecast.Uncertainty.ComputeError != 0 {
		t.Fatalf(
			"expected zero compute error, got %.2f",
			forecast.Uncertainty.ComputeError,
		)
	}
}

func TestForecastRespectsHorizon(t *testing.T) {
	predictor := NewTrendPredictor()

	baseTime := time.Date(
		2026,
		time.September,
		3,
		10,
		0,
		0,
		0,
		time.UTC,
	)

	records := []telemetry.Record{
		{
			Timestamp:          baseTime,
			WorkerID:           "worker-1",
			MemoryUtilization:  40,
			ComputeUtilization: 30,
		},
		{
			Timestamp:          baseTime.Add(1 * time.Second),
			WorkerID:           "worker-1",
			MemoryUtilization:  50,
			ComputeUtilization: 40,
		},
		{
			Timestamp:          baseTime.Add(2 * time.Second),
			WorkerID:           "worker-1",
			MemoryUtilization:  60,
			ComputeUtilization: 50,
		},
	}

	shortForecast, err := predictor.Forecast(
		records,
		1*time.Second,
	)
	if err != nil {
		t.Fatalf(
			"short Forecast returned error: %v",
			err,
		)
	}

	longForecast, err := predictor.Forecast(
		records,
		2*time.Second,
	)
	if err != nil {
		t.Fatalf(
			"long Forecast returned error: %v",
			err,
		)
	}

	if shortForecast.PredictedMemoryUtilization != 70 {
		t.Fatalf(
			"expected 1s memory forecast 70, got %.2f",
			shortForecast.PredictedMemoryUtilization,
		)
	}

	if longForecast.PredictedMemoryUtilization != 80 {
		t.Fatalf(
			"expected 2s memory forecast 80, got %.2f",
			longForecast.PredictedMemoryUtilization,
		)
	}

	if shortForecast.PredictedComputeUtilization != 60 {
		t.Fatalf(
			"expected 1s compute forecast 60, got %.2f",
			shortForecast.PredictedComputeUtilization,
		)
	}

	if longForecast.PredictedComputeUtilization != 70 {
		t.Fatalf(
			"expected 2s compute forecast 70, got %.2f",
			longForecast.PredictedComputeUtilization,
		)
	}
}

func TestForecastRejectsNonIncreasingTimestamps(t *testing.T) {
	predictor := NewTrendPredictor()

	baseTime := time.Date(
		2026,
		time.September,
		3,
		10,
		0,
		0,
		0,
		time.UTC,
	)

	records := []telemetry.Record{
		{
			Timestamp:          baseTime,
			WorkerID:           "worker-1",
			MemoryUtilization:  40,
			ComputeUtilization: 30,
		},
		{
			Timestamp:          baseTime,
			WorkerID:           "worker-1",
			MemoryUtilization:  50,
			ComputeUtilization: 40,
		},
		{
			Timestamp:          baseTime.Add(1 * time.Second),
			WorkerID:           "worker-1",
			MemoryUtilization:  60,
			ComputeUtilization: 50,
		},
	}

	_, err := predictor.Forecast(
		records,
		1*time.Second,
	)

	if !errors.Is(
		err,
		ErrInvalidTelemetryInterval,
	) {
		t.Fatalf(
			"expected ErrInvalidTelemetryInterval, got %v",
			err,
		)
	}
}
