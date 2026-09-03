package simulator

import (
	"context"
	"testing"
	"time"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
)

func TestWorkloadLifecycleCompletesWorkload(t *testing.T) {
	ctx := context.Background()

	store := cluster.NewInMemoryStateStore()

	worker := cluster.Worker{
		ID:                  "worker-1",
		NodeID:              "node-1",
		GPUType:             "NVIDIA-H100",
		TotalMemoryMB:       80000,
		AvailableMemoryMB:   60000,
		MemoryUtilization:   25,
		ComputeUtilization:  40,
		ActiveWorkloadCount: 1,
		State:               cluster.WorkerHealthy,
		LastHeartbeat:       time.Now(),
	}

	if err := store.RegisterWorker(
		ctx,
		worker,
	); err != nil {
		t.Fatal(err)
	}

	workload := cluster.Workload{
		ID:               "workload-1",
		ModelID:          "llama-3-8b",
		ArrivalTime:      time.Now(),
		Priority:         cluster.PriorityStandard,
		RequiredMemoryMB: 20000,
		ExpectedDuration: 10 * time.Second,
		AssignedWorkerID: "worker-1",
		State:            cluster.WorkloadRunning,
	}

	if err := store.CreateWorkload(
		ctx,
		workload,
	); err != nil {
		t.Fatal(err)
	}

	lifecycle := NewWorkloadLifecycle()

	start := time.Now()

	lifecycle.Track(
		workload,
		start,
	)

	early := start.Add(5 * time.Second)

	if lifecycle.Completed(
		workload.ID,
		early,
	) {
		t.Fatal(
			"workload completed before expected duration",
		)
	}

	finished := start.Add(11 * time.Second)

	if !lifecycle.Completed(
		workload.ID,
		finished,
	) {
		t.Fatal(
			"workload did not complete after expected duration",
		)
	}

	simulator := New(store)

	if err := simulator.AdvanceWorkloads(
		ctx,
		lifecycle,
		finished,
	); err != nil {
		t.Fatal(err)
	}

	updatedWorkload, err :=
		store.GetWorkload(
			ctx,
			workload.ID,
		)
	if err != nil {
		t.Fatal(err)
	}

	if updatedWorkload.State !=
		cluster.WorkloadCompleted {
		t.Fatalf(
			"expected workload state COMPLETED, got %s",
			updatedWorkload.State,
		)
	}

	updatedWorker, err :=
		store.GetWorker(
			ctx,
			worker.ID,
		)
	if err != nil {
		t.Fatal(err)
	}

	if updatedWorker.AvailableMemoryMB !=
		worker.TotalMemoryMB {
		t.Fatalf(
			"expected memory to be fully released, got %d MB available",
			updatedWorker.AvailableMemoryMB,
		)
	}

	if updatedWorker.ActiveWorkloadCount != 0 {
		t.Fatalf(
			"expected active workload count 0, got %d",
			updatedWorker.ActiveWorkloadCount,
		)
	}

	if updatedWorker.ComputeUtilization != 30 {
		t.Fatalf(
			"expected compute utilization 30, got %.2f",
			updatedWorker.ComputeUtilization,
		)
	}
}

func TestWorkloadLifecycleTracksRunningWorkload(t *testing.T) {
	workload := cluster.Workload{
		ID:               "workload-1",
		RequiredMemoryMB: 12000,
		ExpectedDuration: 30 * time.Second,
		AssignedWorkerID: "worker-1",
		State:            cluster.WorkloadRunning,
	}

	lifecycle := NewWorkloadLifecycle()

	start := time.Now()

	lifecycle.Track(
		workload,
		start,
	)

	active, ok := lifecycle.activeWorkload(
		workload.ID,
	)

	if !ok {
		t.Fatal(
			"expected workload to be tracked",
		)
	}

	if active.WorkloadID != workload.ID {
		t.Fatalf(
			"expected workload ID %s, got %s",
			workload.ID,
			active.WorkloadID,
		)
	}

	if active.WorkerID !=
		workload.AssignedWorkerID {
		t.Fatalf(
			"expected worker ID %s, got %s",
			workload.AssignedWorkerID,
			active.WorkerID,
		)
	}

	if active.MemoryMB !=
		workload.RequiredMemoryMB {
		t.Fatalf(
			"expected memory %d MB, got %d MB",
			workload.RequiredMemoryMB,
			active.MemoryMB,
		)
	}
}

func TestWorkloadLifecycleRemovesCompletedWorkload(t *testing.T) {
	workload := cluster.Workload{
		ID:               "workload-1",
		RequiredMemoryMB: 8000,
		ExpectedDuration: time.Second,
		AssignedWorkerID: "worker-1",
		State:            cluster.WorkloadRunning,
	}

	lifecycle := NewWorkloadLifecycle()

	lifecycle.Track(
		workload,
		time.Now(),
	)

	_, ok := lifecycle.activeWorkload(
		workload.ID,
	)

	if !ok {
		t.Fatal(
			"expected workload to be tracked",
		)
	}

	lifecycle.Remove(
		workload.ID,
	)

	_, ok = lifecycle.activeWorkload(
		workload.ID,
	)

	if ok {
		t.Fatal(
			"expected workload to be removed",
		)
	}
}
