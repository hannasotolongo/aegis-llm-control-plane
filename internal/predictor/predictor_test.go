package predictor

import (
	"errors"
	"testing"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/telemetry"
)

func TestPredictRisingMemoryContention(t *testing.T) {
	predictor := NewTrendPredictor()

	records := []telemetry.Record{
		{WorkerID: "worker-1", MemoryUtilization: 70, ComputeUtilization: 40},
		{WorkerID: "worker-1", MemoryUtilization: 78, ComputeUtilization: 45},
		{WorkerID: "worker-1", MemoryUtilization: 84, ComputeUtilization: 50},
	}

	prediction, err := predictor.Predict(records)
	if err != nil {
		t.Fatalf("Predict returned error: %v", err)
	}

	if !prediction.PredictedContention {
		t.Fatal("expected predicted contention")
	}

	if prediction.PredictedMemoryUtilization <= 85 {
		t.Fatalf("expected predicted memory above 85, got %.2f", prediction.PredictedMemoryUtilization)
	}

	if prediction.WorkerID != "worker-1" {
		t.Fatalf("expected worker-1, got %q", prediction.WorkerID)
	}
}

func TestPredictStableWorker(t *testing.T) {
	predictor := NewTrendPredictor()

	records := []telemetry.Record{
		{WorkerID: "worker-2", MemoryUtilization: 40, ComputeUtilization: 50},
		{WorkerID: "worker-2", MemoryUtilization: 41, ComputeUtilization: 51},
		{WorkerID: "worker-2", MemoryUtilization: 42, ComputeUtilization: 52},
	}

	prediction, err := predictor.Predict(records)
	if err != nil {
		t.Fatalf("Predict returned error: %v", err)
	}

	if prediction.PredictedContention {
		t.Fatal("did not expect predicted contention")
	}

	if prediction.Confidence < 0.9 {
		t.Fatalf("expected high confidence, got %.2f", prediction.Confidence)
	}
}

func TestPredictInsufficientHistory(t *testing.T) {
	predictor := NewTrendPredictor()

	records := []telemetry.Record{
		{WorkerID: "worker-1", MemoryUtilization: 70},
		{WorkerID: "worker-1", MemoryUtilization: 80},
	}

	_, err := predictor.Predict(records)

	if !errors.Is(err, ErrInsufficientHistory) {
		t.Fatalf("expected ErrInsufficientHistory, got %v", err)
	}
}
