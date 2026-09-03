package predictor

import (
	"testing"
	"time"
)

func TestResultStoreSetAndGet(t *testing.T) {
	store := NewResultStore()

	result := Result{
		Forecast: Forecast{
			WorkerID:                    "worker-1",
			Horizon:                     time.Second,
			PredictedMemoryUtilization:  91,
			PredictedComputeUtilization: 88,
			PredictedContention:         true,
		},
		GeneratedAt: time.Unix(100, 0),
	}

	store.Set(result)

	got, ok := store.Get("worker-1")
	if !ok {
		t.Fatal("expected stored result")
	}

	if got.Forecast.WorkerID != "worker-1" {
		t.Fatalf(
			"unexpected worker ID %q",
			got.Forecast.WorkerID,
		)
	}

	if got.Forecast.PredictedMemoryUtilization != 91 {
		t.Fatalf(
			"expected memory forecast 91, got %.2f",
			got.Forecast.PredictedMemoryUtilization,
		)
	}

	if !got.Forecast.PredictedContention {
		t.Fatal("expected predicted contention")
	}
}

func TestResultStoreReplacesWorkerResult(t *testing.T) {
	store := NewResultStore()

	store.Set(Result{
		Forecast: Forecast{
			WorkerID:                   "worker-1",
			PredictedMemoryUtilization: 50,
		},
	})

	store.Set(Result{
		Forecast: Forecast{
			WorkerID:                   "worker-1",
			PredictedMemoryUtilization: 90,
		},
	})

	result, ok := store.Get("worker-1")
	if !ok {
		t.Fatal("expected stored result")
	}

	if result.Forecast.PredictedMemoryUtilization != 90 {
		t.Fatalf(
			"expected latest memory forecast 90, got %.2f",
			result.Forecast.PredictedMemoryUtilization,
		)
	}
}

func TestResultStoreSnapshotIsIndependent(t *testing.T) {
	store := NewResultStore()

	store.Set(Result{
		Forecast: Forecast{
			WorkerID:                   "worker-1",
			PredictedMemoryUtilization: 80,
		},
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
