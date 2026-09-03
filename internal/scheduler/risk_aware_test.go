package scheduler

import (
	"testing"
	"time"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
	"github.com/hannasotolongo/aegis-llm-control-plane/internal/predictor"
	"github.com/hannasotolongo/aegis-llm-control-plane/internal/risk"
)

type testPredictionProvider struct {
	results map[string]predictor.Result
}

func (p *testPredictionProvider) Get(
	workerID string,
) (predictor.Result, bool) {
	result, ok := p.results[workerID]
	return result, ok
}

func TestSelectWorkerRiskAwarePrefersLowerRiskWorker(t *testing.T) {
	evaluator := risk.NewEvaluator()

	workload := cluster.Workload{
		ID:               "workload-1",
		ModelID:          "llama-3-8b",
		RequiredMemoryMB: 16000,
	}

	workers := []cluster.Worker{
		{
			ID:                  "worker-high-risk",
			State:               cluster.WorkerHealthy,
			TotalMemoryMB:       80000,
			AvailableMemoryMB:   64000,
			MemoryUtilization:   20,
			ComputeUtilization:  20,
			ActiveWorkloadCount: 0,
			CachedModels:        []string{"llama-3-8b"},
		},
		{
			ID:                  "worker-low-risk",
			State:               cluster.WorkerHealthy,
			TotalMemoryMB:       80000,
			AvailableMemoryMB:   56000,
			MemoryUtilization:   30,
			ComputeUtilization:  30,
			ActiveWorkloadCount: 0,
		},
	}

	predictions := &testPredictionProvider{
		results: map[string]predictor.Result{
			"worker-high-risk": {
				GeneratedAt: time.Now(),
				Forecast: predictor.Forecast{
					PredictedMemoryUtilization:  75,
					PredictedComputeUtilization: 80,
					Uncertainty: predictor.Uncertainty{
						MemoryError:  10,
						ComputeError: 10,
						SampleCount:  20,
					},
				},
			},
			"worker-low-risk": {
				GeneratedAt: time.Now(),
				Forecast: predictor.Forecast{
					PredictedMemoryUtilization:  35,
					PredictedComputeUtilization: 40,
					Uncertainty: predictor.Uncertainty{
						MemoryError:  3,
						ComputeError: 3,
						SampleCount:  20,
					},
				},
			},
		},
	}

	selected, err := SelectWorkerRiskAware(
		workload,
		workers,
		predictions,
		evaluator,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if selected.ID != "worker-low-risk" {
		t.Fatalf(
			"expected worker-low-risk, got %s",
			selected.ID,
		)
	}
}

func TestRiskAwareSchedulerFallsBackWithoutPrediction(t *testing.T) {
	evaluator := risk.NewEvaluator()

	workload := cluster.Workload{
		ID:               "workload-1",
		ModelID:          "llama-3-8b",
		RequiredMemoryMB: 8000,
	}

	workers := []cluster.Worker{
		{
			ID:                 "worker-1",
			State:              cluster.WorkerHealthy,
			TotalMemoryMB:      80000,
			AvailableMemoryMB:  60000,
			MemoryUtilization:  20,
			ComputeUtilization: 20,
		},
		{
			ID:                 "worker-2",
			State:              cluster.WorkerHealthy,
			TotalMemoryMB:      80000,
			AvailableMemoryMB:  40000,
			MemoryUtilization:  40,
			ComputeUtilization: 40,
		},
	}

	predictions := &testPredictionProvider{
		results: map[string]predictor.Result{},
	}

	selected, err := SelectWorkerRiskAware(
		workload,
		workers,
		predictions,
		evaluator,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if selected.ID != "worker-1" {
		t.Fatalf(
			"expected baseline fallback to select worker-1, got %s",
			selected.ID,
		)
	}
}
func TestRiskAwareSchedulerFallsBackOnStalePrediction(t *testing.T) {
	evaluator := risk.NewEvaluator()

	workload := cluster.Workload{
		ID:               "workload-1",
		ModelID:          "llama-3-8b",
		RequiredMemoryMB: 8000,
	}

	workers := []cluster.Worker{
		{
			ID:                 "worker-1",
			State:              cluster.WorkerHealthy,
			TotalMemoryMB:      80000,
			AvailableMemoryMB:  60000,
			MemoryUtilization:  20,
			ComputeUtilization: 20,
		},
		{
			ID:                 "worker-2",
			State:              cluster.WorkerHealthy,
			TotalMemoryMB:      80000,
			AvailableMemoryMB:  40000,
			MemoryUtilization:  40,
			ComputeUtilization: 40,
		},
	}

	predictions := &testPredictionProvider{
		results: map[string]predictor.Result{
			"worker-1": {
				GeneratedAt: time.Now().Add(-30 * time.Second),
				Forecast: predictor.Forecast{
					PredictedMemoryUtilization:  95,
					PredictedComputeUtilization: 95,
				},
			},
			"worker-2": {
				GeneratedAt: time.Now().Add(-30 * time.Second),
				Forecast: predictor.Forecast{
					PredictedMemoryUtilization:  10,
					PredictedComputeUtilization: 10,
				},
			},
		},
	}

	selected, err := SelectWorkerRiskAware(
		workload,
		workers,
		predictions,
		evaluator,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if selected.ID != "worker-1" {
		t.Fatalf(
			"expected stale predictions to fall back to worker-1, got %s",
			selected.ID,
		)
	}
}
