package scheduler

import (
	"context"
	"testing"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
)

func TestPlaceWorkload(t *testing.T) {
	ctx := context.Background()
	store := cluster.NewInMemoryStateStore()

	worker := cluster.Worker{
		ID:                "worker-1",
		NodeID:            "node-1",
		GPUType:           "H100",
		TotalMemoryMB:     81920,
		AvailableMemoryMB: 81920,
		State:             cluster.WorkerHealthy,
		TopologyDomain:    "zone-a",
	}

	if err := store.RegisterWorker(ctx, worker); err != nil {
		t.Fatalf("register worker: %v", err)
	}

	workload := cluster.Workload{
		ID:                     "workload-1",
		ModelID:                "llama-3-70b",
		Priority:               cluster.PriorityStandard,
		RequiredMemoryMB:       40960,
		RequiredTopologyDomain: "zone-a",
		State:                  cluster.WorkloadPending,
	}

	if err := store.CreateWorkload(ctx, workload); err != nil {
		t.Fatalf("create workload: %v", err)
	}

	placed, err := PlaceWorkload(ctx, store, workload.ID)
	if err != nil {
		t.Fatalf("PlaceWorkload returned error: %v", err)
	}

	if placed.State != cluster.WorkloadPlaced {
		t.Fatalf("expected state PLACED, got %q", placed.State)
	}

	if placed.AssignedWorkerID != "worker-1" {
		t.Fatalf("expected worker-1, got %q", placed.AssignedWorkerID)
	}

	updatedWorker, err := store.GetWorker(ctx, "worker-1")
	if err != nil {
		t.Fatalf("get worker: %v", err)
	}

	if updatedWorker.AvailableMemoryMB != 40960 {
		t.Fatalf("expected 40960 MB available, got %d", updatedWorker.AvailableMemoryMB)
	}

	if updatedWorker.ActiveWorkloadCount != 1 {
		t.Fatalf("expected 1 active workload, got %d", updatedWorker.ActiveWorkloadCount)
	}
}

func TestPlaceWorkloadWithDecision(t *testing.T) {
	ctx := context.Background()
	store := cluster.NewInMemoryStateStore()

	worker := cluster.Worker{
		ID:                  "worker-1",
		NodeID:              "node-1",
		GPUType:             "H100",
		TotalMemoryMB:       80000,
		AvailableMemoryMB:   60000,
		ComputeUtilization:  20,
		MemoryUtilization:   10,
		ActiveWorkloadCount: 1,
		State:               cluster.WorkerHealthy,
		CachedModels:        []string{"llama-3-70b"},
	}

	if err := store.RegisterWorker(ctx, worker); err != nil {
		t.Fatalf("register worker: %v", err)
	}

	workload := cluster.Workload{
		ID:               "workload-1",
		ModelID:          "llama-3-70b",
		Priority:         cluster.PriorityStandard,
		RequiredMemoryMB: 20000,
		State:            cluster.WorkloadPending,
	}

	if err := store.CreateWorkload(ctx, workload); err != nil {
		t.Fatalf("create workload: %v", err)
	}

	placed, decision, err := PlaceWorkloadWithDecision(
		ctx,
		store,
		workload.ID,
	)
	if err != nil {
		t.Fatalf("PlaceWorkloadWithDecision returned error: %v", err)
	}

	if placed.AssignedWorkerID != worker.ID {
		t.Fatalf(
			"expected workload assigned to %q, got %q",
			worker.ID,
			placed.AssignedWorkerID,
		)
	}

	if decision.WorkloadID != workload.ID {
		t.Fatalf(
			"expected decision workload ID %q, got %q",
			workload.ID,
			decision.WorkloadID,
		)
	}

	if decision.WorkerID != worker.ID {
		t.Fatalf(
			"expected decision worker ID %q, got %q",
			worker.ID,
			decision.WorkerID,
		)
	}

	expected := ExplainWorkerScore(workload, worker)

	if decision.Score != expected {
		t.Fatalf(
			"expected score breakdown %+v, got %+v",
			expected,
			decision.Score,
		)
	}
}

func TestPlaceWorkloadWithDecisionIncludesReasons(t *testing.T) {
	ctx := context.Background()
	store := cluster.NewInMemoryStateStore()

	worker := cluster.Worker{
		ID:                  "worker-1",
		NodeID:              "node-1",
		GPUType:             "H100",
		TotalMemoryMB:       80000,
		AvailableMemoryMB:   60000,
		ComputeUtilization:  20,
		MemoryUtilization:   10,
		ActiveWorkloadCount: 1,
		State:               cluster.WorkerHealthy,
		CachedModels:        []string{"llama-3-70b"},
	}

	if err := store.RegisterWorker(ctx, worker); err != nil {
		t.Fatalf("register worker: %v", err)
	}

	workload := cluster.Workload{
		ID:               "workload-reasons",
		ModelID:          "llama-3-70b",
		Priority:         cluster.PriorityStandard,
		RequiredMemoryMB: 20000,
		State:            cluster.WorkloadPending,
	}

	if err := store.CreateWorkload(ctx, workload); err != nil {
		t.Fatalf("create workload: %v", err)
	}

	_, decision, err := PlaceWorkloadWithDecision(
		ctx,
		store,
		workload.ID,
	)
	if err != nil {
		t.Fatalf("PlaceWorkloadWithDecision returned error: %v", err)
	}

	if len(decision.Reasons) == 0 {
		t.Fatal("expected placement decision to include reasons")
	}

	foundModelLocality := false

	for _, reason := range decision.Reasons {
		if reason == "requested model is already cached on the worker" {
			foundModelLocality = true
			break
		}
	}

	if !foundModelLocality {
		t.Fatalf(
			"expected model locality reason, got %v",
			decision.Reasons,
		)
	}
}
