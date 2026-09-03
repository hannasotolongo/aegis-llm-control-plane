package telemetry

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
)

func TestInMemoryStoreAppendAndList(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	record := Record{
		Timestamp:         time.Now(),
		WorkerID:          "worker-1",
		WorkerState:       cluster.WorkerHealthy,
		AvailableMemoryMB: 60000,
		TotalMemoryMB:     80000,
		WorkloadID:        "workload-1",
		WorkloadState:     cluster.WorkloadRunning,
		ModelID:           "llama-3",
		RequiredMemoryMB:  20000,
		AssignedWorkerID:  "worker-1",
	}

	if err := store.Append(ctx, record); err != nil {
		t.Fatalf("Append returned error: %v", err)
	}

	records, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	if records[0].WorkerID != "worker-1" {
		t.Fatalf(
			"expected worker ID %q, got %q",
			"worker-1",
			records[0].WorkerID,
		)
	}
}

func TestInMemoryStoreConcurrentAppend(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	const recordCount = 100

	var wg sync.WaitGroup

	for i := 0; i < recordCount; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			record := Record{
				Timestamp:         time.Now(),
				WorkerID:          "worker-1",
				WorkerState:       cluster.WorkerHealthy,
				AvailableMemoryMB: 60000,
				TotalMemoryMB:     80000,
			}

			if err := store.Append(ctx, record); err != nil {
				t.Errorf("Append returned error: %v", err)
			}
		}()
	}

	wg.Wait()

	records, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if len(records) != recordCount {
		t.Fatalf(
			"expected %d records, got %d",
			recordCount,
			len(records),
		)
	}
}
