package telemetry

import (
	"errors"
	"testing"
	"time"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
)

func TestApplyDCGMSampleUpdatesWorkerTelemetry(t *testing.T) {
	now := time.Now()

	worker := cluster.Worker{
		ID:                  "worker-1",
		NodeID:              "node-old",
		GPUType:             "H100",
		TotalMemoryMB:       80000,
		AvailableMemoryMB:   64000,
		ComputeUtilization:  20,
		MemoryUtilization:   20,
		ActiveWorkloadCount: 2,
		State:               cluster.WorkerHealthy,
		TopologyDomain:      "zone-a",
		CachedModels: []string{
			"llama-3-8b",
		},
	}

	sample := DCGMSample{
		WorkerID:              worker.ID,
		NodeID:                "node-new",
		GPUType:               "H100",
		Timestamp:             now,
		GPUUtilizationPercent: 72,
		FramebufferTotalMB:    80000,
		FramebufferFreeMB:     20000,
		FramebufferUsedMB:     60000,
	}

	updated, err := ApplyDCGMSample(
		worker,
		sample,
		now,
		DefaultDCGMMaxAge,
	)
	if err != nil {
		t.Fatalf(
			"apply DCGM sample: %v",
			err,
		)
	}

	if updated.ComputeUtilization != 72 {
		t.Fatalf(
			"expected compute utilization 72, got %.2f",
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
			"expected 20000 MB available, got %d",
			updated.AvailableMemoryMB,
		)
	}

	if updated.ActiveWorkloadCount != worker.ActiveWorkloadCount {
		t.Fatalf(
			"telemetry overwrote active workload count: before=%d after=%d",
			worker.ActiveWorkloadCount,
			updated.ActiveWorkloadCount,
		)
	}

	if updated.TopologyDomain != worker.TopologyDomain {
		t.Fatalf(
			"telemetry overwrote topology domain: before=%q after=%q",
			worker.TopologyDomain,
			updated.TopologyDomain,
		)
	}

	if len(updated.CachedModels) != 1 ||
		updated.CachedModels[0] != "llama-3-8b" {
		t.Fatalf(
			"telemetry overwrote cached models: %v",
			updated.CachedModels,
		)
	}
}

func TestApplyDCGMSampleRejectsStaleTelemetry(t *testing.T) {
	now := time.Now()

	worker := cluster.Worker{
		ID:    "worker-1",
		State: cluster.WorkerHealthy,
	}

	sample := DCGMSample{
		WorkerID:              worker.ID,
		Timestamp:             now.Add(-30 * time.Second),
		GPUUtilizationPercent: 50,
		FramebufferTotalMB:    80000,
		FramebufferFreeMB:     40000,
		FramebufferUsedMB:     40000,
	}

	_, err := ApplyDCGMSample(
		worker,
		sample,
		now,
		DefaultDCGMMaxAge,
	)

	if err == nil {
		t.Fatal(
			"expected stale DCGM sample to be rejected",
		)
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
}

func TestApplyDCGMSampleRejectsInvalidGPUUtilization(t *testing.T) {
	now := time.Now()

	worker := cluster.Worker{
		ID:    "worker-1",
		State: cluster.WorkerHealthy,
	}

	sample := DCGMSample{
		WorkerID:              worker.ID,
		Timestamp:             now,
		GPUUtilizationPercent: 140,
		FramebufferTotalMB:    80000,
		FramebufferFreeMB:     40000,
		FramebufferUsedMB:     40000,
	}

	_, err := ApplyDCGMSample(
		worker,
		sample,
		now,
		DefaultDCGMMaxAge,
	)

	if err == nil {
		t.Fatal(
			"expected invalid GPU utilization to be rejected",
		)
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
}

func TestApplyDCGMSampleRejectsInconsistentMemory(t *testing.T) {
	now := time.Now()

	worker := cluster.Worker{
		ID:    "worker-1",
		State: cluster.WorkerHealthy,
	}

	sample := DCGMSample{
		WorkerID:              worker.ID,
		Timestamp:             now,
		GPUUtilizationPercent: 50,
		FramebufferTotalMB:    80000,
		FramebufferFreeMB:     50000,
		FramebufferUsedMB:     40000,
	}

	_, err := ApplyDCGMSample(
		worker,
		sample,
		now,
		DefaultDCGMMaxAge,
	)

	if err == nil {
		t.Fatal(
			"expected inconsistent framebuffer memory to be rejected",
		)
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
}

func TestApplyDCGMSampleRejectsWorkerMismatch(t *testing.T) {
	now := time.Now()

	worker := cluster.Worker{
		ID:    "worker-1",
		State: cluster.WorkerHealthy,
	}

	sample := DCGMSample{
		WorkerID:              "worker-2",
		Timestamp:             now,
		GPUUtilizationPercent: 50,
		FramebufferTotalMB:    80000,
		FramebufferFreeMB:     40000,
		FramebufferUsedMB:     40000,
	}

	_, err := ApplyDCGMSample(
		worker,
		sample,
		now,
		DefaultDCGMMaxAge,
	)

	if err == nil {
		t.Fatal(
			"expected mismatched worker sample to be rejected",
		)
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
}
