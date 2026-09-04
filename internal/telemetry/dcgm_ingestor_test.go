package telemetry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
)

func TestDCGMIngestorUpdatesWorkerForValidSample(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	store := cluster.NewInMemoryStateStore()

	worker := cluster.Worker{
		ID:                  "worker-1",
		NodeID:              "node-1",
		GPUType:             "H100",
		TotalMemoryMB:       80000,
		AvailableMemoryMB:   60000,
		ComputeUtilization:  20,
		MemoryUtilization:   25,
		ActiveWorkloadCount: 1,
		State:               cluster.WorkerHealthy,
		LastHeartbeat:       now.Add(-5 * time.Second),
		TopologyDomain:      "zone-a",
	}

	if err := store.RegisterWorker(ctx, worker); err != nil {
		t.Fatalf("register worker: %v", err)
	}

	ingestor := NewDCGMIngestor(
		store,
		DefaultDCGMMaxAge,
	)

	sample := DCGMSample{
		WorkerID:              worker.ID,
		NodeID:                worker.NodeID,
		GPUType:               worker.GPUType,
		Timestamp:             now,
		GPUUtilizationPercent: 70,
		FramebufferTotalMB:    80000,
		FramebufferFreeMB:     20000,
		FramebufferUsedMB:     60000,
	}

	if err := ingestor.Ingest(
		ctx,
		sample,
		now,
	); err != nil {
		t.Fatalf("ingest DCGM sample: %v", err)
	}

	updated, err := store.GetWorker(
		ctx,
		worker.ID,
	)
	if err != nil {
		t.Fatalf("get worker: %v", err)
	}

	if updated.ComputeUtilization != 70 {
		t.Fatalf(
			"expected compute utilization 70, got %.2f",
			updated.ComputeUtilization,
		)
	}

	if updated.MemoryUtilization != 75 {
		t.Fatalf(
			"expected memory utilization 75, got %.2f",
			updated.MemoryUtilization,
		)
	}

	if updated.AvailableMemoryMB != 20000 {
		t.Fatalf(
			"expected available memory 20000, got %d",
			updated.AvailableMemoryMB,
		)
	}

	if !updated.LastHeartbeat.Equal(now) {
		t.Fatalf(
			"expected heartbeat %v, got %v",
			now,
			updated.LastHeartbeat,
		)
	}
}

func TestDCGMIngestorRejectsStaleSampleWithoutMutatingWorker(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	store := cluster.NewInMemoryStateStore()

	worker := cluster.Worker{
		ID:                  "worker-1",
		NodeID:              "node-1",
		GPUType:             "H100",
		TotalMemoryMB:       80000,
		AvailableMemoryMB:   60000,
		ComputeUtilization:  20,
		MemoryUtilization:   25,
		ActiveWorkloadCount: 1,
		State:               cluster.WorkerHealthy,
		LastHeartbeat:       now.Add(-5 * time.Second),
		TopologyDomain:      "zone-a",
	}

	if err := store.RegisterWorker(ctx, worker); err != nil {
		t.Fatalf("register worker: %v", err)
	}

	ingestor := NewDCGMIngestor(
		store,
		DefaultDCGMMaxAge,
	)

	sample := DCGMSample{
		WorkerID:              worker.ID,
		NodeID:                worker.NodeID,
		GPUType:               worker.GPUType,
		Timestamp:             now.Add(-30 * time.Second),
		GPUUtilizationPercent: 99,
		FramebufferTotalMB:    80000,
		FramebufferFreeMB:     1000,
		FramebufferUsedMB:     79000,
	}

	err := ingestor.Ingest(
		ctx,
		sample,
		now,
	)

	if err == nil {
		t.Fatal("expected stale telemetry error")
	}

	if !errors.Is(
		err,
		ErrStaleDCGMSample,
	) {
		t.Fatalf(
			"expected ErrStaleDCGMSample, got %v",
			err,
		)
	}

	after, err := store.GetWorker(
		ctx,
		worker.ID,
	)
	if err != nil {
		t.Fatalf("get worker: %v", err)
	}

	if after.ComputeUtilization != worker.ComputeUtilization {
		t.Fatalf(
			"stale telemetry mutated compute utilization: before=%.2f after=%.2f",
			worker.ComputeUtilization,
			after.ComputeUtilization,
		)
	}

	if after.AvailableMemoryMB != worker.AvailableMemoryMB {
		t.Fatalf(
			"stale telemetry mutated available memory: before=%d after=%d",
			worker.AvailableMemoryMB,
			after.AvailableMemoryMB,
		)
	}

	if !after.LastHeartbeat.Equal(worker.LastHeartbeat) {
		t.Fatalf(
			"stale telemetry mutated heartbeat: before=%v after=%v",
			worker.LastHeartbeat,
			after.LastHeartbeat,
		)
	}
}

func TestDCGMIngestorRejectsInvalidSampleWithoutMutatingWorker(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	store := cluster.NewInMemoryStateStore()

	worker := cluster.Worker{
		ID:                  "worker-1",
		NodeID:              "node-1",
		GPUType:             "H100",
		TotalMemoryMB:       80000,
		AvailableMemoryMB:   60000,
		ComputeUtilization:  20,
		MemoryUtilization:   25,
		ActiveWorkloadCount: 1,
		State:               cluster.WorkerHealthy,
		LastHeartbeat:       now.Add(-5 * time.Second),
		TopologyDomain:      "zone-a",
	}

	if err := store.RegisterWorker(ctx, worker); err != nil {
		t.Fatalf("register worker: %v", err)
	}

	ingestor := NewDCGMIngestor(
		store,
		DefaultDCGMMaxAge,
	)

	sample := DCGMSample{
		WorkerID:              worker.ID,
		NodeID:                worker.NodeID,
		GPUType:               worker.GPUType,
		Timestamp:             now,
		GPUUtilizationPercent: 150,
		FramebufferTotalMB:    80000,
		FramebufferFreeMB:     20000,
		FramebufferUsedMB:     60000,
	}

	err := ingestor.Ingest(
		ctx,
		sample,
		now,
	)

	if err == nil {
		t.Fatal("expected invalid telemetry error")
	}

	if !errors.Is(
		err,
		ErrInvalidDCGMSample,
	) {
		t.Fatalf(
			"expected ErrInvalidDCGMSample, got %v",
			err,
		)
	}

	after, err := store.GetWorker(
		ctx,
		worker.ID,
	)
	if err != nil {
		t.Fatalf("get worker: %v", err)
	}

	if after.ComputeUtilization != worker.ComputeUtilization {
		t.Fatalf(
			"invalid telemetry mutated compute utilization: before=%.2f after=%.2f",
			worker.ComputeUtilization,
			after.ComputeUtilization,
		)
	}

	if after.AvailableMemoryMB != worker.AvailableMemoryMB {
		t.Fatalf(
			"invalid telemetry mutated available memory: before=%d after=%d",
			worker.AvailableMemoryMB,
			after.AvailableMemoryMB,
		)
	}
}

func TestDCGMIngestorReturnsWorkerNotFound(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	store := cluster.NewInMemoryStateStore()

	ingestor := NewDCGMIngestor(
		store,
		DefaultDCGMMaxAge,
	)

	sample := DCGMSample{
		WorkerID:              "missing-worker",
		Timestamp:             now,
		GPUUtilizationPercent: 50,
		FramebufferTotalMB:    80000,
		FramebufferFreeMB:     40000,
		FramebufferUsedMB:     40000,
	}

	err := ingestor.Ingest(
		ctx,
		sample,
		now,
	)

	if err == nil {
		t.Fatal("expected worker not found error")
	}

	if !errors.Is(
		err,
		cluster.ErrWorkerNotFound,
	) {
		t.Fatalf(
			"expected ErrWorkerNotFound, got %v",
			err,
		)
	}
}
