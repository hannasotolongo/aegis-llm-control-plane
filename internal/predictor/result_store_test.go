package predictor

import (
	"testing"
	"time"
)

func TestResultStoreSetAndGet(t *testing.T) {
	store := NewResultStore()

	result := Result{
		Prediction: Prediction{
			WorkerID:                    "worker-1",
			PredictedMemoryUtilization:  91,
			PredictedComputeUtilization: 88,
			PredictedContention:         true,
			Confidence:                  0.95,
		},
		Decision: Decision{
			Mode:                DecisionPredictive,
			TrustPrediction:     true,
			PredictedContention: true,
			Confidence:          0.95,
		},
		GeneratedAt: time.Unix(100, 0),
	}

	store.Set(result)

	got, ok := store.Get("worker-1")
	if !ok {
		t.Fatal("expected stored result")
	}

	if got.Prediction.WorkerID != "worker-1" {
		t.Fatalf("unexpected worker ID %q", got.Prediction.WorkerID)
	}

	if !got.Decision.TrustPrediction {
		t.Fatal("expected trusted prediction")
	}
}

func TestResultStoreReplacesWorkerResult(t *testing.T) {
	store := NewResultStore()

	store.Set(Result{
		Prediction: Prediction{WorkerID: "worker-1", Confidence: 0.50},
	})

	store.Set(Result{
		Prediction: Prediction{WorkerID: "worker-1", Confidence: 0.90},
	})

	result, ok := store.Get("worker-1")
	if !ok {
		t.Fatal("expected stored result")
	}

	if result.Prediction.Confidence != 0.90 {
		t.Fatalf("expected latest confidence 0.90, got %.2f", result.Prediction.Confidence)
	}
}

func TestResultStoreSnapshotIsIndependent(t *testing.T) {
	store := NewResultStore()
	store.Set(Result{
		Prediction: Prediction{WorkerID: "worker-1", Confidence: 0.80},
	})

	snapshot := store.Snapshot()
	delete(snapshot, "worker-1")

	if _, ok := store.Get("worker-1"); !ok {
		t.Fatal("snapshot mutation changed result store")
	}
}

func TestResultStoreIgnoresEmptyWorkerID(t *testing.T) {
	store := NewResultStore()
	store.Set(Result{})

	if len(store.Snapshot()) != 0 {
		t.Fatal("expected empty worker ID to be ignored")
	}
}
