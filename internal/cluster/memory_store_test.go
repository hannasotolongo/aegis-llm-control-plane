package cluster

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
)

func TestRegisterWorkerStoresValidWorker(t *testing.T) {
	store := NewInMemoryStateStore()

	worker := Worker{
		ID:                 "worker-1",
		NodeID:             "node-1",
		TotalMemoryMB:      80000,
		AvailableMemoryMB:  64000,
		ComputeUtilization: 20,
		MemoryUtilization:  18,
		State:              WorkerHealthy,
	}

	if err := store.RegisterWorker(context.Background(), worker); err != nil {
		t.Fatalf("expected worker registration to succeed, got: %v", err)
	}

	storedWorker, err := store.GetWorker(context.Background(), "worker-1")
	if err != nil {
		t.Fatalf("expected worker-1 to exist in store: %v", err)
	}

	if storedWorker.ID != worker.ID {
		t.Fatalf(
			"expected stored worker ID %q, got %q",
			worker.ID,
			storedWorker.ID,
		)
	}
}

func TestRegisterWorkerRejectsDuplicateID(t *testing.T) {
	store := NewInMemoryStateStore()

	worker := Worker{
		ID:                "worker-1",
		NodeID:            "node-1",
		TotalMemoryMB:     80000,
		AvailableMemoryMB: 64000,
		State:             WorkerHealthy,
	}

	if err := store.RegisterWorker(context.Background(), worker); err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	err := store.RegisterWorker(context.Background(), worker)

	if err == nil {
		t.Fatal("expected duplicate worker registration to fail")
	}

	if !errors.Is(err, ErrWorkerAlreadyExists) {
		t.Fatalf(
			"expected ErrWorkerAlreadyExists, got: %v",
			err,
		)
	}
}

func TestRegisterWorkerRejectsCanceledContext(t *testing.T) {
	store := NewInMemoryStateStore()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	worker := Worker{
		ID:                "worker-1",
		NodeID:            "node-1",
		TotalMemoryMB:     80000,
		AvailableMemoryMB: 64000,
		State:             WorkerHealthy,
	}

	if err := store.RegisterWorker(ctx, worker); err == nil {
		t.Fatal("expected registration with canceled context to fail")
	}
}

func TestRegisterWorkerConcurrent(t *testing.T) {
	store := NewInMemoryStateStore()

	const workerCount = 100

	var wg sync.WaitGroup
	wg.Add(workerCount)

	for i := 0; i < workerCount; i++ {
		go func(id int) {
			defer wg.Done()

			worker := Worker{
				ID:                fmt.Sprintf("worker-%d", id),
				NodeID:            fmt.Sprintf("node-%d", id),
				TotalMemoryMB:     80000,
				AvailableMemoryMB: 64000,
				State:             WorkerHealthy,
			}

			if err := store.RegisterWorker(context.Background(), worker); err != nil {
				t.Errorf("worker registration failed: %v", err)
			}
		}(i)
	}

	wg.Wait()

	workers, err := store.ListWorkers(context.Background())
	if err != nil {
		t.Fatalf("failed to list workers: %v", err)
	}

	if len(workers) != workerCount {
		t.Fatalf(
			"expected %d registered workers, got %d",
			workerCount,
			len(workers),
		)
	}
}

func TestGetWorkerReturnsRegisteredWorker(t *testing.T) {
	store := NewInMemoryStateStore()

	worker := Worker{
		ID:                 "worker-1",
		NodeID:             "node-1",
		TotalMemoryMB:      80000,
		AvailableMemoryMB:  64000,
		ComputeUtilization: 30,
		MemoryUtilization:  22,
		State:              WorkerHealthy,
	}

	if err := store.RegisterWorker(context.Background(), worker); err != nil {
		t.Fatalf("worker registration failed: %v", err)
	}

	storedWorker, err := store.GetWorker(context.Background(), "worker-1")
	if err != nil {
		t.Fatalf("expected GetWorker to succeed, got: %v", err)
	}

	if storedWorker.ID != worker.ID {
		t.Fatalf(
			"expected worker ID %q, got %q",
			worker.ID,
			storedWorker.ID,
		)
	}

	if storedWorker.AvailableMemoryMB != worker.AvailableMemoryMB {
		t.Fatalf(
			"expected available memory %d MB, got %d MB",
			worker.AvailableMemoryMB,
			storedWorker.AvailableMemoryMB,
		)
	}
}

func TestGetWorkerReturnsTypedNotFoundError(t *testing.T) {
	store := NewInMemoryStateStore()

	_, err := store.GetWorker(
		context.Background(),
		"missing-worker",
	)

	if err == nil {
		t.Fatal("expected GetWorker to fail for unknown worker")
	}

	if !errors.Is(err, ErrWorkerNotFound) {
		t.Fatalf(
			"expected ErrWorkerNotFound, got: %v",
			err,
		)
	}
}

func TestListWorkersReturnsRegisteredWorkers(t *testing.T) {
	store := NewInMemoryStateStore()

	workers := []Worker{
		{
			ID:                "worker-1",
			NodeID:            "node-1",
			TotalMemoryMB:     80000,
			AvailableMemoryMB: 64000,
			State:             WorkerHealthy,
		},
		{
			ID:                "worker-2",
			NodeID:            "node-2",
			TotalMemoryMB:     80000,
			AvailableMemoryMB: 48000,
			State:             WorkerHealthy,
		},
	}

	for _, worker := range workers {
		if err := store.RegisterWorker(context.Background(), worker); err != nil {
			t.Fatalf("worker registration failed: %v", err)
		}
	}

	storedWorkers, err := store.ListWorkers(context.Background())
	if err != nil {
		t.Fatalf("expected ListWorkers to succeed, got: %v", err)
	}

	if len(storedWorkers) != len(workers) {
		t.Fatalf(
			"expected %d workers, got %d",
			len(workers),
			len(storedWorkers),
		)
	}
}

func TestUpdateWorkerChangesWorkerState(t *testing.T) {
	store := NewInMemoryStateStore()

	worker := Worker{
		ID:                 "worker-1",
		NodeID:             "node-1",
		TotalMemoryMB:      80000,
		AvailableMemoryMB:  64000,
		ComputeUtilization: 20,
		MemoryUtilization:  18,
		State:              WorkerHealthy,
	}

	if err := store.RegisterWorker(context.Background(), worker); err != nil {
		t.Fatalf("worker registration failed: %v", err)
	}

	worker.AvailableMemoryMB = 32000
	worker.ComputeUtilization = 75
	worker.MemoryUtilization = 60
	worker.State = WorkerSuspected

	if err := store.UpdateWorker(context.Background(), worker); err != nil {
		t.Fatalf("expected worker update to succeed, got: %v", err)
	}

	updatedWorker, err := store.GetWorker(context.Background(), "worker-1")
	if err != nil {
		t.Fatalf("failed to retrieve updated worker: %v", err)
	}

	if updatedWorker.AvailableMemoryMB != 32000 {
		t.Fatalf(
			"expected available memory 32000 MB, got %d MB",
			updatedWorker.AvailableMemoryMB,
		)
	}

	if updatedWorker.ComputeUtilization != 75 {
		t.Fatalf(
			"expected compute utilization 75, got %.2f",
			updatedWorker.ComputeUtilization,
		)
	}

	if updatedWorker.State != WorkerSuspected {
		t.Fatalf(
			"expected worker state %q, got %q",
			WorkerSuspected,
			updatedWorker.State,
		)
	}
}

func TestUpdateWorkerReturnsTypedNotFoundError(t *testing.T) {
	store := NewInMemoryStateStore()

	worker := Worker{
		ID:                "missing-worker",
		NodeID:            "node-1",
		TotalMemoryMB:     80000,
		AvailableMemoryMB: 64000,
		State:             WorkerHealthy,
	}

	err := store.UpdateWorker(context.Background(), worker)

	if err == nil {
		t.Fatal("expected update of unknown worker to fail")
	}

	if !errors.Is(err, ErrWorkerNotFound) {
		t.Fatalf(
			"expected ErrWorkerNotFound, got: %v",
			err,
		)
	}
}

func TestRegisterWorkerDefensivelyCopiesCachedModels(t *testing.T) {
	store := NewInMemoryStateStore()

	cachedModels := []string{
		"llama-3",
		"mistral",
	}

	worker := Worker{
		ID:                "worker-1",
		NodeID:            "node-1",
		TotalMemoryMB:     80000,
		AvailableMemoryMB: 64000,
		State:             WorkerHealthy,
		CachedModels:      cachedModels,
	}

	if err := store.RegisterWorker(context.Background(), worker); err != nil {
		t.Fatalf("worker registration failed: %v", err)
	}

	cachedModels[0] = "modified-outside-store"

	storedWorker, err := store.GetWorker(context.Background(), "worker-1")
	if err != nil {
		t.Fatalf("failed to retrieve worker: %v", err)
	}

	if storedWorker.CachedModels[0] != "llama-3" {
		t.Fatalf(
			"stored CachedModels was modified through caller-owned slice: %q",
			storedWorker.CachedModels[0],
		)
	}
}

func TestGetWorkerReturnsDefensiveCopy(t *testing.T) {
	store := NewInMemoryStateStore()

	worker := Worker{
		ID:                "worker-1",
		NodeID:            "node-1",
		TotalMemoryMB:     80000,
		AvailableMemoryMB: 64000,
		State:             WorkerHealthy,
		CachedModels: []string{
			"llama-3",
			"mistral",
		},
	}

	if err := store.RegisterWorker(context.Background(), worker); err != nil {
		t.Fatalf("worker registration failed: %v", err)
	}

	firstRead, err := store.GetWorker(context.Background(), "worker-1")
	if err != nil {
		t.Fatalf("failed to retrieve worker: %v", err)
	}

	firstRead.CachedModels[0] = "modified-by-reader"

	secondRead, err := store.GetWorker(context.Background(), "worker-1")
	if err != nil {
		t.Fatalf("failed to retrieve worker a second time: %v", err)
	}

	if secondRead.CachedModels[0] != "llama-3" {
		t.Fatalf(
			"stored CachedModels was modified through returned worker: %q",
			secondRead.CachedModels[0],
		)
	}
}

func TestConcurrentWorkerReadsAndUpdates(t *testing.T) {
	store := NewInMemoryStateStore()

	worker := Worker{
		ID:                "worker-1",
		NodeID:            "node-1",
		TotalMemoryMB:     80000,
		AvailableMemoryMB: 64000,
		State:             WorkerHealthy,
	}

	if err := store.RegisterWorker(context.Background(), worker); err != nil {
		t.Fatalf("worker registration failed: %v", err)
	}

	const operationCount = 100

	var wg sync.WaitGroup
	wg.Add(operationCount * 2)

	for i := 0; i < operationCount; i++ {
		go func() {
			defer wg.Done()

			if _, err := store.GetWorker(
				context.Background(),
				"worker-1",
			); err != nil {
				t.Errorf("GetWorker failed: %v", err)
			}
		}()

		go func(utilization float64) {
			defer wg.Done()

			updatedWorker := worker
			updatedWorker.ComputeUtilization = utilization

			if err := store.UpdateWorker(
				context.Background(),
				updatedWorker,
			); err != nil {
				t.Errorf("UpdateWorker failed: %v", err)
			}
		}(float64(i % 100))
	}

	wg.Wait()
}

func TestCommitPlacementPreventsConcurrentOvercommit(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryStateStore()

	worker := Worker{
		ID:                  "worker-1",
		NodeID:              "node-1",
		GPUType:             "H100",
		TotalMemoryMB:       20000,
		AvailableMemoryMB:   20000,
		ActiveWorkloadCount: 0,
		State:               WorkerHealthy,
	}

	if err := store.RegisterWorker(ctx, worker); err != nil {
		t.Fatalf("register worker: %v", err)
	}

	workload1 := Workload{
		ID:               "workload-1",
		ModelID:          "model-1",
		Priority:         PriorityStandard,
		RequiredMemoryMB: 15000,
		State:            WorkloadPending,
	}

	workload2 := Workload{
		ID:               "workload-2",
		ModelID:          "model-2",
		Priority:         PriorityStandard,
		RequiredMemoryMB: 15000,
		State:            WorkloadPending,
	}

	if err := store.CreateWorkload(ctx, workload1); err != nil {
		t.Fatalf("create workload 1: %v", err)
	}

	if err := store.CreateWorkload(ctx, workload2); err != nil {
		t.Fatalf("create workload 2: %v", err)
	}

	var wg sync.WaitGroup
	results := make(chan error, 2)

	for _, workloadID := range []string{
		workload1.ID,
		workload2.ID,
	} {
		wg.Add(1)

		go func(id string) {
			defer wg.Done()

			_, err := store.CommitPlacement(
				ctx,
				id,
				worker.ID,
			)

			results <- err
		}(workloadID)
	}

	wg.Wait()
	close(results)

	successes := 0
	conflicts := 0

	for err := range results {
		if err == nil {
			successes++
			continue
		}

		if errors.Is(err, ErrPlacementConflict) {
			conflicts++
			continue
		}

		t.Fatalf("unexpected placement error: %v", err)
	}

	if successes != 1 {
		t.Fatalf(
			"expected exactly 1 successful placement, got %d",
			successes,
		)
	}

	if conflicts != 1 {
		t.Fatalf(
			"expected exactly 1 placement conflict, got %d",
			conflicts,
		)
	}

	updatedWorker, err := store.GetWorker(ctx, worker.ID)
	if err != nil {
		t.Fatalf("get worker: %v", err)
	}

	if updatedWorker.AvailableMemoryMB != 5000 {
		t.Fatalf(
			"expected 5000 MB available, got %d",
			updatedWorker.AvailableMemoryMB,
		)
	}

	if updatedWorker.ActiveWorkloadCount != 1 {
		t.Fatalf(
			"expected 1 active workload, got %d",
			updatedWorker.ActiveWorkloadCount,
		)
	}
}

func TestReleasePlacementRestoresWorkerResources(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryStateStore()

	worker := Worker{
		ID:                  "worker-release",
		NodeID:              "node-release",
		GPUType:             "H100",
		TotalMemoryMB:       80000,
		AvailableMemoryMB:   60000,
		ActiveWorkloadCount: 0,
		State:               WorkerHealthy,
	}

	workload := Workload{
		ID:               "workload-release",
		ModelID:          "model-1",
		Priority:         PriorityStandard,
		RequiredMemoryMB: 20000,
		State:            WorkloadPending,
	}

	if err := store.RegisterWorker(ctx, worker); err != nil {
		t.Fatalf("register worker: %v", err)
	}

	if err := store.CreateWorkload(ctx, workload); err != nil {
		t.Fatalf("create workload: %v", err)
	}

	_, err := store.CommitPlacement(
		ctx,
		workload.ID,
		worker.ID,
	)
	if err != nil {
		t.Fatalf("commit placement: %v", err)
	}

	workerAfterPlacement, err := store.GetWorker(ctx, worker.ID)
	if err != nil {
		t.Fatalf("get worker after placement: %v", err)
	}

	if workerAfterPlacement.AvailableMemoryMB != 40000 {
		t.Fatalf(
			"expected 40000 MB available after placement, got %d",
			workerAfterPlacement.AvailableMemoryMB,
		)
	}

	if workerAfterPlacement.ActiveWorkloadCount != 1 {
		t.Fatalf(
			"expected 1 active workload after placement, got %d",
			workerAfterPlacement.ActiveWorkloadCount,
		)
	}

	released, err := store.ReleasePlacement(
		ctx,
		workload.ID,
		WorkloadRecovering,
	)
	if err != nil {
		t.Fatalf("release placement: %v", err)
	}

	if released.State != WorkloadRecovering {
		t.Fatalf(
			"expected recovering workload, got %q",
			released.State,
		)
	}

	if released.AssignedWorkerID != "" {
		t.Fatalf(
			"expected assignment to be cleared, got %q",
			released.AssignedWorkerID,
		)
	}

	workerAfterRelease, err := store.GetWorker(ctx, worker.ID)
	if err != nil {
		t.Fatalf("get worker after release: %v", err)
	}

	if workerAfterRelease.AvailableMemoryMB != 60000 {
		t.Fatalf(
			"expected 60000 MB available after release, got %d",
			workerAfterRelease.AvailableMemoryMB,
		)
	}

	if workerAfterRelease.ActiveWorkloadCount != 0 {
		t.Fatalf(
			"expected 0 active workloads after release, got %d",
			workerAfterRelease.ActiveWorkloadCount,
		)
	}
}
func TestCommitPlacementRejectsWorkerStateChange(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryStateStore()

	worker := Worker{
		ID:                  "worker-1",
		NodeID:              "node-1",
		GPUType:             "H100",
		TotalMemoryMB:       80000,
		AvailableMemoryMB:   64000,
		MemoryUtilization:   20,
		ComputeUtilization:  20,
		ActiveWorkloadCount: 0,
		State:               WorkerHealthy,
	}

	workload := Workload{
		ID:               "workload-1",
		ModelID:          "llama-3-8b",
		Priority:         PriorityStandard,
		RequiredMemoryMB: 16000,
		State:            WorkloadPending,
	}

	if err := store.RegisterWorker(
		ctx,
		worker,
	); err != nil {
		t.Fatalf(
			"register worker: %v",
			err,
		)
	}

	if err := store.CreateWorkload(
		ctx,
		workload,
	); err != nil {
		t.Fatalf(
			"create workload: %v",
			err,
		)
	}

	// Simulate a scheduler selecting the worker while
	// it is healthy, followed by a state change before
	// placement is committed.
	updatedWorker, err :=
		store.GetWorker(
			ctx,
			worker.ID,
		)
	if err != nil {
		t.Fatalf(
			"get worker: %v",
			err,
		)
	}

	updatedWorker.State =
		WorkerUnhealthy

	if err := store.UpdateWorker(
		ctx,
		updatedWorker,
	); err != nil {
		t.Fatalf(
			"update worker: %v",
			err,
		)
	}

	_, err = store.CommitPlacement(
		ctx,
		workload.ID,
		worker.ID,
	)

	if err == nil {
		t.Fatal(
			"expected placement conflict after worker became unhealthy",
		)
	}

	if !errors.Is(
		err,
		ErrPlacementConflict,
	) {
		t.Fatalf(
			"expected ErrPlacementConflict, got %v",
			err,
		)
	}

	storedWorker, err :=
		store.GetWorker(
			ctx,
			worker.ID,
		)
	if err != nil {
		t.Fatalf(
			"get worker after failed placement: %v",
			err,
		)
	}

	if storedWorker.AvailableMemoryMB !=
		worker.AvailableMemoryMB {
		t.Fatalf(
			"worker memory changed after rejected placement: before=%d after=%d",
			worker.AvailableMemoryMB,
			storedWorker.AvailableMemoryMB,
		)
	}

	if storedWorker.ActiveWorkloadCount != 0 {
		t.Fatalf(
			"worker active workload count changed after rejected placement: %d",
			storedWorker.ActiveWorkloadCount,
		)
	}

	storedWorkload, err :=
		store.GetWorkload(
			ctx,
			workload.ID,
		)
	if err != nil {
		t.Fatalf(
			"get workload after failed placement: %v",
			err,
		)
	}

	if storedWorkload.State != WorkloadPending {
		t.Fatalf(
			"workload state changed after rejected placement: %s",
			storedWorkload.State,
		)
	}

	if storedWorkload.AssignedWorkerID != "" {
		t.Fatalf(
			"workload was assigned after rejected placement: %s",
			storedWorkload.AssignedWorkerID,
		)
	}
}
