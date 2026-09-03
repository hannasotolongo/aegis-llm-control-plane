package scheduler

import (
	"context"
	"errors"
	"testing"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
)

type commitPlacementErrorStore struct {
	cluster.StateStore
	err error
}

func (s *commitPlacementErrorStore) CommitPlacement(
	ctx context.Context,
	workloadID string,
	workerID string,
) (cluster.Workload, error) {
	return cluster.Workload{}, s.err
}

func TestSchedulePending(t *testing.T) {
	ctx := context.Background()
	store := cluster.NewInMemoryStateStore()
	service := NewService(store)

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

	batch := cluster.Workload{
		ID:               "batch-workload",
		ModelID:          "model-a",
		Priority:         cluster.PriorityBatch,
		RequiredMemoryMB: 10000,
		State:            cluster.WorkloadPending,
	}

	critical := cluster.Workload{
		ID:               "critical-workload",
		ModelID:          "model-a",
		Priority:         cluster.PriorityCritical,
		RequiredMemoryMB: 10000,
		State:            cluster.WorkloadPending,
	}

	if err := store.CreateWorkload(ctx, batch); err != nil {
		t.Fatalf("create batch workload: %v", err)
	}

	if err := store.CreateWorkload(ctx, critical); err != nil {
		t.Fatalf("create critical workload: %v", err)
	}

	placed, err := service.SchedulePending(ctx)
	if err != nil {
		t.Fatalf("SchedulePending returned error: %v", err)
	}

	if len(placed) != 2 {
		t.Fatalf("expected 2 placed workloads, got %d", len(placed))
	}

	if placed[0].ID != "critical-workload" {
		t.Fatalf("expected critical workload first, got %q", placed[0].ID)
	}

	if placed[1].ID != "batch-workload" {
		t.Fatalf("expected batch workload second, got %q", placed[1].ID)
	}

	updatedWorker, err := store.GetWorker(ctx, "worker-1")
	if err != nil {
		t.Fatalf("get worker: %v", err)
	}

	if updatedWorker.AvailableMemoryMB != 61920 {
		t.Fatalf(
			"expected 61920 MB available, got %d",
			updatedWorker.AvailableMemoryMB,
		)
	}

	if updatedWorker.ActiveWorkloadCount != 2 {
		t.Fatalf(
			"expected 2 active workloads, got %d",
			updatedWorker.ActiveWorkloadCount,
		)
	}
}

func TestSchedulePendingReturnsUnexpectedPlacementError(t *testing.T) {
	ctx := context.Background()
	baseStore := cluster.NewInMemoryStateStore()

	worker := cluster.Worker{
		ID:                "worker-1",
		NodeID:            "node-1",
		GPUType:           "H100",
		TotalMemoryMB:     81920,
		AvailableMemoryMB: 81920,
		State:             cluster.WorkerHealthy,
		TopologyDomain:    "zone-a",
	}

	if err := baseStore.RegisterWorker(ctx, worker); err != nil {
		t.Fatalf("register worker: %v", err)
	}

	workload := cluster.Workload{
		ID:               "workload-1",
		ModelID:          "model-a",
		Priority:         cluster.PriorityStandard,
		RequiredMemoryMB: 10000,
		State:            cluster.WorkloadPending,
	}

	if err := baseStore.CreateWorkload(ctx, workload); err != nil {
		t.Fatalf("create workload: %v", err)
	}

	expectedErr := errors.New("simulated placement commit failure")

	store := &commitPlacementErrorStore{
		StateStore: baseStore,
		err:        expectedErr,
	}

	service := NewService(store)

	placed, err := service.SchedulePending(ctx)

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}

	if len(placed) != 0 {
		t.Fatalf("expected 0 placed workloads, got %d", len(placed))
	}
}
