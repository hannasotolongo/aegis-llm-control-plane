package scheduler

import (
	"testing"
	"time"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
	"github.com/hannasotolongo/aegis-llm-control-plane/internal/predictor"
)

func TestPredictiveSelectionAvoidsForecastContention(t *testing.T) {
	workers := []cluster.Worker{
		{
			ID:                 "worker-1",
			State:              cluster.WorkerHealthy,
			AvailableMemoryMB:  70000,
			ComputeUtilization: 20,
			MemoryUtilization:  20,
		},
		{
			ID:                 "worker-2",
			State:              cluster.WorkerHealthy,
			AvailableMemoryMB:  65000,
			ComputeUtilization: 30,
			MemoryUtilization:  30,
		},
	}

	workload := cluster.Workload{
		ID:               "workload-1",
		ModelID:          "model-a",
		RequiredMemoryMB: 10000,
	}

	results := predictor.NewResultStore()

	results.Set(predictor.Result{
		Prediction: predictor.Prediction{
			WorkerID:                    "worker-1",
			PredictedMemoryUtilization:  95,
			PredictedComputeUtilization: 95,
			PredictedContention:         true,
			Confidence:                  0.95,
		},
		Decision: predictor.Decision{
			Mode:                predictor.DecisionPredictive,
			TrustPrediction:     true,
			PredictedContention: true,
			Confidence:          0.95,
		},
		GeneratedAt: time.Now(),
	})

	selected, err := SelectWorkerPredictive(workload, workers, results)
	if err != nil {
		t.Fatalf("select worker: %v", err)
	}

	if selected.ID != "worker-2" {
		t.Fatalf("expected worker-2, got %s", selected.ID)
	}
}

func TestPredictiveSelectionFallsBackWhenPredictionUntrusted(t *testing.T) {
	workers := []cluster.Worker{
		{
			ID:                 "worker-1",
			State:              cluster.WorkerHealthy,
			AvailableMemoryMB:  70000,
			ComputeUtilization: 20,
			MemoryUtilization:  20,
		},
		{
			ID:                 "worker-2",
			State:              cluster.WorkerHealthy,
			AvailableMemoryMB:  60000,
			ComputeUtilization: 40,
			MemoryUtilization:  40,
		},
	}

	workload := cluster.Workload{
		ID:               "workload-1",
		ModelID:          "model-a",
		RequiredMemoryMB: 10000,
	}

	results := predictor.NewResultStore()

	results.Set(predictor.Result{
		Prediction: predictor.Prediction{
			WorkerID:                    "worker-1",
			PredictedMemoryUtilization:  99,
			PredictedComputeUtilization: 99,
			PredictedContention:         true,
			Confidence:                  0.20,
		},
		Decision: predictor.Decision{
			Mode:            predictor.DecisionDeterministic,
			TrustPrediction: false,
			Confidence:      0.20,
		},
	})

	selected, err := SelectWorkerPredictive(workload, workers, results)
	if err != nil {
		t.Fatalf("select worker: %v", err)
	}

	if selected.ID != "worker-1" {
		t.Fatalf("expected deterministic fallback to worker-1, got %s", selected.ID)
	}
}

func TestPredictiveSelectionWithoutPredictionMatchesBaseline(t *testing.T) {
	workers := []cluster.Worker{
		{
			ID:                 "worker-1",
			State:              cluster.WorkerHealthy,
			AvailableMemoryMB:  70000,
			ComputeUtilization: 20,
			MemoryUtilization:  20,
		},
		{
			ID:                 "worker-2",
			State:              cluster.WorkerHealthy,
			AvailableMemoryMB:  60000,
			ComputeUtilization: 40,
			MemoryUtilization:  40,
		},
	}

	workload := cluster.Workload{
		ID:               "workload-1",
		ModelID:          "model-a",
		RequiredMemoryMB: 10000,
	}

	baseline, err := SelectWorker(workload, workers)
	if err != nil {
		t.Fatalf("baseline selection: %v", err)
	}

	predictive, err := SelectWorkerPredictive(workload, workers, nil)
	if err != nil {
		t.Fatalf("predictive selection: %v", err)
	}

	if predictive.ID != baseline.ID {
		t.Fatalf("expected baseline worker %s, got %s", baseline.ID, predictive.ID)
	}
}

func TestPredictiveSelectionRejectsStalePrediction(t *testing.T) {
	workers := []cluster.Worker{
		{
			ID:                 "worker-1",
			State:              cluster.WorkerHealthy,
			AvailableMemoryMB:  70000,
			ComputeUtilization: 20,
			MemoryUtilization:  20,
		},
		{
			ID:                 "worker-2",
			State:              cluster.WorkerHealthy,
			AvailableMemoryMB:  60000,
			ComputeUtilization: 40,
			MemoryUtilization:  40,
		},
	}

	workload := cluster.Workload{
		ID:               "workload-1",
		ModelID:          "model-a",
		RequiredMemoryMB: 10000,
	}

	results := predictor.NewResultStore()

	results.Set(predictor.Result{
		Prediction: predictor.Prediction{
			WorkerID:                    "worker-1",
			PredictedMemoryUtilization:  99,
			PredictedComputeUtilization: 99,
			PredictedContention:         true,
			Confidence:                  0.90,
		},
		Decision: predictor.Decision{
			Mode:            predictor.DecisionPredictive,
			TrustPrediction: true,
			Confidence:      0.90,
		},
		GeneratedAt: time.Now().Add(-30 * time.Second),
	})

	selected, err := SelectWorkerPredictive(workload, workers, results)
	if err != nil {
		t.Fatalf("select worker: %v", err)
	}

	if selected.ID != "worker-1" {
		t.Fatalf("expected stale prediction fallback to worker-1, got %s", selected.ID)
	}
}
