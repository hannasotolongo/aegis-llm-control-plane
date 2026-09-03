package cluster

import (
	"context"
	"testing"
	"time"
)

func TestReconcileClusterRecoversCheckpointableWorkload(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryStateStore()

	now := time.Now()

	worker := Worker{
		ID:                  "worker-1",
		NodeID:              "node-1",
		GPUType:             "H100",
		TotalMemoryMB:       80000,
		AvailableMemoryMB:   40000,
		ActiveWorkloadCount: 1,
		State:               WorkerHealthy,
		LastHeartbeat:       now.Add(-1 * time.Minute),
	}

	if err := store.RegisterWorker(ctx, worker); err != nil {
		t.Fatalf("RegisterWorker returned error: %v", err)
	}

	workload := Workload{
		ID:               "workload-1",
		ModelID:          "llama-3-70b",
		Priority:         PriorityStandard,
		RequiredMemoryMB: 20000,
		Checkpointable:   true,
		State:            WorkloadPlaced,
		AssignedWorkerID: "worker-1",
	}

	if err := store.CreateWorkload(ctx, workload); err != nil {
		t.Fatalf("CreateWorkload returned error: %v", err)
	}

	config := WorkerHealthConfig{
		SuspectedAfter: 10 * time.Second,
		UnhealthyAfter: 30 * time.Second,
	}

	result, err := ReconcileCluster(
		ctx,
		store,
		now,
		config,
	)
	if err != nil {
		t.Fatalf("ReconcileCluster returned error: %v", err)
	}

	if result.UpdatedWorkers != 1 {
		t.Fatalf(
			"expected 1 updated worker, got %d",
			result.UpdatedWorkers,
		)
	}

	if result.RecoveredWorkloads != 1 {
		t.Fatalf(
			"expected 1 recovered workload, got %d",
			result.RecoveredWorkloads,
		)
	}

	updatedWorker, err := store.GetWorker(ctx, "worker-1")
	if err != nil {
		t.Fatalf("GetWorker returned error: %v", err)
	}

	if updatedWorker.State != WorkerUnhealthy {
		t.Fatalf(
			"expected worker state %q, got %q",
			WorkerUnhealthy,
			updatedWorker.State,
		)
	}

	if updatedWorker.AvailableMemoryMB != 60000 {
		t.Fatalf(
			"expected 60000 MB available after recovery, got %d",
			updatedWorker.AvailableMemoryMB,
		)
	}

	if updatedWorker.ActiveWorkloadCount != 0 {
		t.Fatalf(
			"expected 0 active workloads after recovery, got %d",
			updatedWorker.ActiveWorkloadCount,
		)
	}

	updatedWorkload, err := store.GetWorkload(ctx, "workload-1")
	if err != nil {
		t.Fatalf("GetWorkload returned error: %v", err)
	}

	if updatedWorkload.State != WorkloadRecovering {
		t.Fatalf(
			"expected workload state %q, got %q",
			WorkloadRecovering,
			updatedWorkload.State,
		)
	}

	if updatedWorkload.AssignedWorkerID != "" {
		t.Fatalf(
			"expected recovered workload to have no assigned worker, got %q",
			updatedWorkload.AssignedWorkerID,
		)
	}
}

func TestReconcileClusterFailsNonCheckpointableWorkload(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryStateStore()

	now := time.Now()

	worker := Worker{
		ID:                  "worker-1",
		NodeID:              "node-1",
		GPUType:             "H100",
		TotalMemoryMB:       80000,
		AvailableMemoryMB:   40000,
		ActiveWorkloadCount: 1,
		State:               WorkerHealthy,
		LastHeartbeat:       now.Add(-1 * time.Minute),
	}

	if err := store.RegisterWorker(ctx, worker); err != nil {
		t.Fatalf("RegisterWorker returned error: %v", err)
	}

	workload := Workload{
		ID:               "workload-1",
		ModelID:          "llama-3-70b",
		Priority:         PriorityStandard,
		RequiredMemoryMB: 20000,
		Checkpointable:   false,
		State:            WorkloadRunning,
		AssignedWorkerID: "worker-1",
	}

	if err := store.CreateWorkload(ctx, workload); err != nil {
		t.Fatalf("CreateWorkload returned error: %v", err)
	}

	config := WorkerHealthConfig{
		SuspectedAfter: 10 * time.Second,
		UnhealthyAfter: 30 * time.Second,
	}

	result, err := ReconcileCluster(
		ctx,
		store,
		now,
		config,
	)
	if err != nil {
		t.Fatalf("ReconcileCluster returned error: %v", err)
	}

	if result.FailedWorkloads != 1 {
		t.Fatalf(
			"expected 1 failed workload, got %d",
			result.FailedWorkloads,
		)
	}

	updatedWorker, err := store.GetWorker(ctx, "worker-1")
	if err != nil {
		t.Fatalf("GetWorker returned error: %v", err)
	}

	if updatedWorker.AvailableMemoryMB != 60000 {
		t.Fatalf(
			"expected 60000 MB available after failure, got %d",
			updatedWorker.AvailableMemoryMB,
		)
	}

	if updatedWorker.ActiveWorkloadCount != 0 {
		t.Fatalf(
			"expected 0 active workloads after failure, got %d",
			updatedWorker.ActiveWorkloadCount,
		)
	}

	updatedWorkload, err := store.GetWorkload(ctx, "workload-1")
	if err != nil {
		t.Fatalf("GetWorkload returned error: %v", err)
	}

	if updatedWorkload.State != WorkloadFailed {
		t.Fatalf(
			"expected workload state %q, got %q",
			WorkloadFailed,
			updatedWorkload.State,
		)
	}

	if updatedWorkload.AssignedWorkerID != "" {
		t.Fatalf(
			"expected failed workload to have no assigned worker, got %q",
			updatedWorkload.AssignedWorkerID,
		)
	}
}
