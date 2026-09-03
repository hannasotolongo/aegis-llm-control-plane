package predictor

import (
	"context"
	"testing"
	"time"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/telemetry"
)

type testTelemetryStore struct {
	records []telemetry.Record
}

func (s *testTelemetryStore) List(
	ctx context.Context,
) ([]telemetry.Record, error) {
	return append(
		[]telemetry.Record(nil),
		s.records...,
	), nil
}

func TestServiceProcessesOnlyNewTelemetry(t *testing.T) {
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

	store := &testTelemetryStore{
		records: []telemetry.Record{
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
		},
	}

	results := NewResultStore()

	service := NewService(
		store,
		NewTrendPredictor(),
		results,
		1*time.Second,
		1*time.Second,
	)

	err := service.predict(context.Background())
	if err != nil {
		t.Fatalf(
			"first prediction failed: %v",
			err,
		)
	}

	if service.processedRecords != 3 {
		t.Fatalf(
			"expected 3 processed records, got %d",
			service.processedRecords,
		)
	}

	recent := service.history.Recent(
		"worker-1",
		16,
	)

	if len(recent) != 3 {
		t.Fatalf(
			"expected 3 history records, got %d",
			len(recent),
		)
	}

	result, exists := results.Get("worker-1")
	if !exists {
		t.Fatal(
			"expected forecast result for worker-1",
		)
	}

	if result.Forecast.WorkerID != "worker-1" {
		t.Fatalf(
			"expected worker-1, got %q",
			result.Forecast.WorkerID,
		)
	}

	if result.Forecast.Horizon != 1*time.Second {
		t.Fatalf(
			"expected 1s horizon, got %s",
			result.Forecast.Horizon,
		)
	}

	if result.Forecast.PredictedMemoryUtilization != 70 {
		t.Fatalf(
			"expected predicted memory utilization 70, got %.2f",
			result.Forecast.PredictedMemoryUtilization,
		)
	}

	if result.Forecast.PredictedComputeUtilization != 60 {
		t.Fatalf(
			"expected predicted compute utilization 60, got %.2f",
			result.Forecast.PredictedComputeUtilization,
		)
	}

	store.records = append(
		store.records,
		telemetry.Record{
			Timestamp:          baseTime.Add(3 * time.Second),
			WorkerID:           "worker-1",
			MemoryUtilization:  70,
			ComputeUtilization: 60,
		},
	)

	err = service.predict(context.Background())
	if err != nil {
		t.Fatalf(
			"second prediction failed: %v",
			err,
		)
	}

	if service.processedRecords != 4 {
		t.Fatalf(
			"expected 4 processed records, got %d",
			service.processedRecords,
		)
	}

	recent = service.history.Recent(
		"worker-1",
		16,
	)

	if len(recent) != 4 {
		t.Fatalf(
			"expected 4 history records, got %d",
			len(recent),
		)
	}

	result, exists = results.Get("worker-1")
	if !exists {
		t.Fatal(
			"expected updated forecast result for worker-1",
		)
	}

	if result.Forecast.PredictedMemoryUtilization != 80 {
		t.Fatalf(
			"expected predicted memory utilization 80, got %.2f",
			result.Forecast.PredictedMemoryUtilization,
		)
	}

	if result.Forecast.PredictedComputeUtilization != 70 {
		t.Fatalf(
			"expected predicted compute utilization 70, got %.2f",
			result.Forecast.PredictedComputeUtilization,
		)
	}

	if result.Forecast.Uncertainty.SampleCount != 1 {
		t.Fatalf(
			"expected 1 uncertainty sample, got %d",
			result.Forecast.Uncertainty.SampleCount,
		)
	}
}
