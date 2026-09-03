package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
	"github.com/hannasotolongo/aegis-llm-control-plane/internal/predictor"
)

func TestPredictiveServiceChangesPlacementFromBaseline(t *testing.T) {
	ctx := context.Background()
	store := cluster.NewInMemoryStateStore()

	workers := []cluster.Worker{
		{
			ID:                 "worker-1",
			NodeID:             "node-1",
			GPUType:            "NVIDIA-H100",
			TotalMemoryMB:      81920,
			AvailableMemoryMB:  70000,
			ComputeUtilization: 20,
			MemoryUtilization:  20,
			State:              cluster.WorkerHealthy,
			LastHeartbeat:      time.Now(),
			TopologyDomain:     "rack-1",
		},
		{
			ID:                 "worker-2",
			NodeID:             "node-2",
			GPUType:            "NVIDIA-H100",
			TotalMemoryMB:      81920,
			AvailableMemoryMB:  65000,
			ComputeUtilization: 30,
			MemoryUtilization:  30,
			State:              cluster.WorkerHealthy,
			LastHeartbeat:      time.Now(),
			TopologyDomain:     "rack-1",
		},
	}

	for _, worker := range workers {
		if err := store.RegisterWorker(ctx, worker); err != nil {
			t.Fatalf("register worker: %v", err)
		}
	}

	workload := cluster.Workload{
		ID:               "workload-1",
		ModelID:          "model-a",
		ArrivalTime:      time.Now(),
		Priority:         cluster.PriorityStandard,
		RequiredMemoryMB: 10000,
		EstimatedCompute: 0.50,
		ExpectedDuration: time.Minute,
		LatencySLO:       time.Second,
		Checkpointable:   true,
		State:            cluster.WorkloadQueued,
	}

	if err := store.CreateWorkload(ctx, workload); err != nil {
		t.Fatalf("create workload: %v", err)
	}

	baseline, err := SelectWorker(workload, workers)
	if err != nil {
		t.Fatalf("baseline selection: %v", err)
	}

	if baseline.ID != "worker-1" {
		t.Fatalf(
			"expected baseline worker-1, got %s",
			baseline.ID,
		)
	}

	results := predictor.NewResultStore()

	results.Set(predictor.Result{
		Forecast: predictor.Forecast{
			WorkerID:                    "worker-1",
			Horizon:                     time.Second,
			PredictedMemoryUtilization:  97,
			PredictedComputeUtilization: 95,
			PredictedContention:         true,
		},
		GeneratedAt: time.Now(),
	})

	service := NewPredictiveService(
		store,
		results,
	)

	placed, err := service.SchedulePending(ctx)
	if err != nil {
		t.Fatalf("schedule pending: %v", err)
	}

	if len(placed) != 1 {
		t.Fatalf(
			"expected one placement, got %d",
			len(placed),
		)
	}

	if placed[0].AssignedWorkerID != "worker-2" {
		t.Fatalf(
			"expected predictive placement on worker-2, got %s",
			placed[0].AssignedWorkerID,
		)
	}

	worker2, err := store.GetWorker(
		ctx,
		"worker-2",
	)
	if err != nil {
		t.Fatalf("get worker-2: %v", err)
	}

	if worker2.AvailableMemoryMB != 55000 {
		t.Fatalf(
			"expected worker-2 memory reservation to leave 55000 MB, got %d",
			worker2.AvailableMemoryMB,
		)
	}

	if worker2.ActiveWorkloadCount != 1 {
		t.Fatalf(
			"expected worker-2 active workload count 1, got %d",
			worker2.ActiveWorkloadCount,
		)
	}
}
