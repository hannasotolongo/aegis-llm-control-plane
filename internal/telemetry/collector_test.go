package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
)

func TestCollectorRecordsSnapshot(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryStore()
	collector := NewCollector(store)

	timestamp := time.Now()

	snapshot := cluster.Snapshot{
		Workers: []cluster.Worker{
			{
				ID:                  "worker-1",
				TotalMemoryMB:       80000,
				AvailableMemoryMB:   60000,
				ComputeUtilization:  45,
				MemoryUtilization:   25,
				ActiveWorkloadCount: 1,
				State:               cluster.WorkerHealthy,
			},
		},
	}

	snapshot.Workloads = []cluster.Workload{
		{
			ID:               "workload-1",
			ModelID:          "llama-3",
			Priority:         cluster.PriorityStandard,
			RequiredMemoryMB: 20000,
			EstimatedCompute: 0.70,
			LatencySLO:       2 * time.Second,
			Checkpointable:   true,
			State:            cluster.WorkloadRunning,
			AssignedWorkerID: "worker-1",
		},
	}

	if err := collector.RecordSnapshot(
		ctx,
		snapshot,
		timestamp,
	); err != nil {
		t.Fatalf("RecordSnapshot returned error: %v", err)
	}

	records, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 telemetry record, got %d", len(records))
	}

	record := records[0]

	if record.WorkerID != "worker-1" {
		t.Fatalf("expected worker-1, got %q", record.WorkerID)
	}

	if record.WorkloadID != "workload-1" {
		t.Fatalf("expected workload-1, got %q", record.WorkloadID)
	}

	if record.ModelID != "llama-3" {
		t.Fatalf("expected llama-3, got %q", record.ModelID)
	}

	if record.RequiredMemoryMB != 20000 {
		t.Fatalf(
			"expected required memory 20000, got %d",
			record.RequiredMemoryMB,
		)
	}

	if !record.Timestamp.Equal(timestamp) {
		t.Fatalf(
			"expected timestamp %v, got %v",
			timestamp,
			record.Timestamp,
		)
	}
}

func TestCollectorRecordsIdleWorker(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryStore()
	collector := NewCollector(store)

	timestamp := time.Now()

	snapshot := cluster.Snapshot{
		Workers: []cluster.Worker{
			{
				ID:                  "worker-idle",
				TotalMemoryMB:       80000,
				AvailableMemoryMB:   80000,
				ComputeUtilization:  0,
				MemoryUtilization:   0,
				ActiveWorkloadCount: 0,
				State:               cluster.WorkerHealthy,
			},
		},
	}

	if err := collector.RecordSnapshot(
		ctx,
		snapshot,
		timestamp,
	); err != nil {
		t.Fatalf("RecordSnapshot returned error: %v", err)
	}

	records, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 telemetry record, got %d", len(records))
	}

	record := records[0]

	if record.WorkerID != "worker-idle" {
		t.Fatalf("expected worker-idle, got %q", record.WorkerID)
	}

	if record.WorkloadID != "" {
		t.Fatalf("expected no workload ID, got %q", record.WorkloadID)
	}

	if record.AvailableMemoryMB != 80000 {
		t.Fatalf(
			"expected available memory 80000, got %d",
			record.AvailableMemoryMB,
		)
	}

	if record.ActiveWorkloadCount != 0 {
		t.Fatalf(
			"expected 0 active workloads, got %d",
			record.ActiveWorkloadCount,
		)
	}
}
