package scheduler

import (
	"errors"
	"testing"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
)

func TestSelectWorker(t *testing.T) {
	workload := cluster.Workload{
		ID:                     "workload-1",
		ModelID:                "llama-3-70b",
		Priority:               cluster.PriorityStandard,
		RequiredMemoryMB:       40000,
		RequiredTopologyDomain: "zone-a",
		State:                  cluster.WorkloadPending,
	}

	workers := []cluster.Worker{
		{
			ID:                 "worker-1",
			AvailableMemoryMB:  60000,
			ComputeUtilization: 70,
			State:              cluster.WorkerHealthy,
			TopologyDomain:     "zone-a",
		},
		{
			ID:                 "worker-2",
			AvailableMemoryMB:  70000,
			ComputeUtilization: 25,
			State:              cluster.WorkerHealthy,
			TopologyDomain:     "zone-a",
		},
	}

	selected, err := SelectWorker(workload, workers)
	if err != nil {
		t.Fatalf("SelectWorker returned error: %v", err)
	}

	if selected.ID != "worker-2" {
		t.Fatalf("expected worker-2, got %q", selected.ID)
	}
}

func TestSelectWorkerRejectsInsufficientMemory(t *testing.T) {
	workload := cluster.Workload{
		ID:               "workload-1",
		RequiredMemoryMB: 90000,
	}

	workers := []cluster.Worker{
		{
			ID:                "worker-1",
			AvailableMemoryMB: 81920,
			State:             cluster.WorkerHealthy,
		},
	}

	_, err := SelectWorker(workload, workers)
	if !errors.Is(err, ErrNoEligibleWorker) {
		t.Fatalf("expected ErrNoEligibleWorker, got %v", err)
	}
}

func TestSelectWorkerRejectsUnhealthyWorker(t *testing.T) {
	workload := cluster.Workload{
		ID:               "workload-1",
		RequiredMemoryMB: 1000,
	}

	workers := []cluster.Worker{
		{
			ID:                "worker-1",
			AvailableMemoryMB: 81920,
			State:             cluster.WorkerUnhealthy,
		},
	}

	_, err := SelectWorker(workload, workers)
	if !errors.Is(err, ErrNoEligibleWorker) {
		t.Fatalf("expected ErrNoEligibleWorker, got %v", err)
	}
}

func TestSelectWorkerPrefersLowerUtilization(t *testing.T) {
	workload := cluster.Workload{
		ID:               "workload-1",
		ModelID:          "llama-3-70b",
		RequiredMemoryMB: 20000,
		State:            cluster.WorkloadPending,
	}

	workers := []cluster.Worker{
		{
			ID:                 "worker-busy",
			AvailableMemoryMB:  60000,
			ComputeUtilization: 80,
			MemoryUtilization:  70,
			State:              cluster.WorkerHealthy,
		},
		{
			ID:                 "worker-idle",
			AvailableMemoryMB:  60000,
			ComputeUtilization: 20,
			MemoryUtilization:  20,
			State:              cluster.WorkerHealthy,
		},
	}

	selected, err := SelectWorker(workload, workers)
	if err != nil {
		t.Fatalf("SelectWorker returned error: %v", err)
	}

	if selected.ID != "worker-idle" {
		t.Fatalf(
			"expected worker-idle due to lower utilization, got %q",
			selected.ID,
		)
	}
}

func TestSelectWorkerPrefersFewerActiveWorkloads(t *testing.T) {
	workload := cluster.Workload{
		ID:               "workload-1",
		ModelID:          "llama-3-70b",
		RequiredMemoryMB: 20000,
		State:            cluster.WorkloadPending,
	}

	workers := []cluster.Worker{
		{
			ID:                  "worker-loaded",
			AvailableMemoryMB:   60000,
			ComputeUtilization:  30,
			MemoryUtilization:   30,
			ActiveWorkloadCount: 5,
			State:               cluster.WorkerHealthy,
		},
		{
			ID:                  "worker-light",
			AvailableMemoryMB:   60000,
			ComputeUtilization:  30,
			MemoryUtilization:   30,
			ActiveWorkloadCount: 1,
			State:               cluster.WorkerHealthy,
		},
	}

	selected, err := SelectWorker(workload, workers)
	if err != nil {
		t.Fatalf("SelectWorker returned error: %v", err)
	}

	if selected.ID != "worker-light" {
		t.Fatalf(
			"expected worker-light due to fewer active workloads, got %q",
			selected.ID,
		)
	}
}

func TestExplainWorkerScore(t *testing.T) {
	workload := cluster.Workload{
		ID:      "workload-1",
		ModelID: "llama-3-70b",
	}

	worker := cluster.Worker{
		ID:                  "worker-1",
		AvailableMemoryMB:   51200,
		ComputeUtilization:  20,
		MemoryUtilization:   10,
		ActiveWorkloadCount: 2,
		CachedModels:        []string{"llama-3-70b"},
	}

	breakdown := ExplainWorkerScore(workload, worker)

	if breakdown.MemoryHeadroom != 50 {
		t.Fatalf(
			"expected memory headroom 50, got %v",
			breakdown.MemoryHeadroom,
		)
	}

	if breakdown.ComputePenalty != 20 {
		t.Fatalf(
			"expected compute penalty 20, got %v",
			breakdown.ComputePenalty,
		)
	}

	if breakdown.MemoryPenalty != 10 {
		t.Fatalf(
			"expected memory penalty 10, got %v",
			breakdown.MemoryPenalty,
		)
	}

	if breakdown.ActiveWorkloadPenalty != 20 {
		t.Fatalf(
			"expected active workload penalty 20, got %v",
			breakdown.ActiveWorkloadPenalty,
		)
	}

	if breakdown.ModelLocalityBonus != 100 {
		t.Fatalf(
			"expected model locality bonus 100, got %v",
			breakdown.ModelLocalityBonus,
		)
	}

	if breakdown.Total != 100 {
		t.Fatalf(
			"expected total score 100, got %v",
			breakdown.Total,
		)
	}
}
