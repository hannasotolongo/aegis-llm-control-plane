package cluster

import (
	"testing"
	"time"
)

func TestValidateWorkerAcceptsValidWorker(t *testing.T) {
	worker := Worker{
		ID:                 "worker-1",
		NodeID:             "node-1",
		TotalMemoryMB:      80000,
		AvailableMemoryMB:  64000,
		ComputeUtilization: 25,
		MemoryUtilization:  20,
		State:              WorkerHealthy,
	}

	if err := ValidateWorker(worker); err != nil {
		t.Fatalf("expected valid worker, got error: %v", err)
	}
}

func TestValidateWorkerRejectsMissingID(t *testing.T) {
	worker := Worker{
		NodeID:            "node-1",
		TotalMemoryMB:     80000,
		AvailableMemoryMB: 64000,
	}

	if err := ValidateWorker(worker); err == nil {
		t.Fatal("expected error for missing worker ID")
	}
}

func TestValidateWorkerRejectsExcessAvailableMemory(t *testing.T) {
	worker := Worker{
		ID:                "worker-1",
		NodeID:            "node-1",
		TotalMemoryMB:     80000,
		AvailableMemoryMB: 90000,
	}

	if err := ValidateWorker(worker); err == nil {
		t.Fatal("expected error when available memory exceeds total memory")
	}
}

func TestValidateWorkerRejectsInvalidComputeUtilization(t *testing.T) {
	worker := Worker{
		ID:                 "worker-1",
		NodeID:             "node-1",
		TotalMemoryMB:      80000,
		AvailableMemoryMB:  64000,
		ComputeUtilization: 125,
	}

	if err := ValidateWorker(worker); err == nil {
		t.Fatal("expected error for compute utilization above 100")
	}
}

func TestValidateWorkloadAcceptsValidPendingWorkload(t *testing.T) {
	workload := Workload{
		ID:               "workload-1",
		ModelID:          "llama-3",
		ArrivalTime:      time.Now(),
		Priority:         PriorityStandard,
		PromptTokens:     1200,
		MaxOutputTokens:  400,
		BatchSize:        1,
		RequiredMemoryMB: 16000,
		KVCacheMemoryMB:  2000,
		EstimatedCompute: 45,
		ExpectedDuration: 2 * time.Second,
		LatencySLO:       500 * time.Millisecond,
		Checkpointable:   true,
		State:            WorkloadPending,
	}

	if err := ValidateWorkload(workload); err != nil {
		t.Fatalf(
			"expected valid workload, got error: %v",
			err,
		)
	}
}

func TestValidateWorkloadRejectsMissingID(t *testing.T) {
	workload := Workload{
		ModelID:          "llama-3",
		Priority:         PriorityStandard,
		RequiredMemoryMB: 16000,
		State:            WorkloadPending,
	}

	if err := ValidateWorkload(workload); err == nil {
		t.Fatal("expected error for missing workload ID")
	}
}

func TestValidateWorkloadRejectsMissingModelID(t *testing.T) {
	workload := Workload{
		ID:               "workload-1",
		Priority:         PriorityStandard,
		RequiredMemoryMB: 16000,
		State:            WorkloadPending,
	}

	if err := ValidateWorkload(workload); err == nil {
		t.Fatal("expected error for missing model ID")
	}
}

func TestValidateWorkloadRejectsZeroMemoryRequirement(t *testing.T) {
	workload := Workload{
		ID:       "workload-1",
		ModelID:  "llama-3",
		Priority: PriorityStandard,
		State:    WorkloadPending,
	}

	if err := ValidateWorkload(workload); err == nil {
		t.Fatal("expected error for zero GPU memory requirement")
	}
}

func TestValidateWorkloadRejectsNegativePromptTokens(t *testing.T) {
	workload := Workload{
		ID:               "workload-1",
		ModelID:          "llama-3",
		Priority:         PriorityStandard,
		PromptTokens:     -1,
		RequiredMemoryMB: 16000,
		State:            WorkloadPending,
	}

	if err := ValidateWorkload(workload); err == nil {
		t.Fatal("expected error for negative prompt tokens")
	}
}

func TestValidateWorkloadRejectsNegativeMaxOutputTokens(t *testing.T) {
	workload := Workload{
		ID:               "workload-1",
		ModelID:          "llama-3",
		Priority:         PriorityStandard,
		MaxOutputTokens:  -1,
		RequiredMemoryMB: 16000,
		State:            WorkloadPending,
	}

	if err := ValidateWorkload(workload); err == nil {
		t.Fatal("expected error for negative max output tokens")
	}
}

func TestValidateWorkloadRejectsNegativeBatchSize(t *testing.T) {
	workload := Workload{
		ID:               "workload-1",
		ModelID:          "llama-3",
		Priority:         PriorityStandard,
		BatchSize:        -1,
		RequiredMemoryMB: 16000,
		State:            WorkloadPending,
	}

	if err := ValidateWorkload(workload); err == nil {
		t.Fatal("expected error for negative batch size")
	}
}

func TestValidateWorkloadRejectsKVCacheLargerThanTotalMemory(t *testing.T) {
	workload := Workload{
		ID:               "workload-1",
		ModelID:          "llama-3",
		Priority:         PriorityStandard,
		RequiredMemoryMB: 8000,
		KVCacheMemoryMB:  12000,
		State:            WorkloadPending,
	}

	if err := ValidateWorkload(workload); err == nil {
		t.Fatal(
			"expected error when KV cache memory exceeds required memory",
		)
	}
}

func TestValidateWorkloadRejectsNegativeCompute(t *testing.T) {
	workload := Workload{
		ID:               "workload-1",
		ModelID:          "llama-3",
		Priority:         PriorityStandard,
		RequiredMemoryMB: 16000,
		EstimatedCompute: -1,
		State:            WorkloadPending,
	}

	if err := ValidateWorkload(workload); err == nil {
		t.Fatal("expected error for negative estimated compute")
	}
}

func TestValidateWorkloadRejectsInvalidPriority(t *testing.T) {
	workload := Workload{
		ID:               "workload-1",
		ModelID:          "llama-3",
		Priority:         Priority("INVALID"),
		RequiredMemoryMB: 16000,
		State:            WorkloadPending,
	}

	if err := ValidateWorkload(workload); err == nil {
		t.Fatal("expected error for invalid workload priority")
	}
}

func TestValidateWorkloadRejectsInvalidState(t *testing.T) {
	workload := Workload{
		ID:               "workload-1",
		ModelID:          "llama-3",
		Priority:         PriorityStandard,
		RequiredMemoryMB: 16000,
		State:            WorkloadState("UNKNOWN"),
	}

	if err := ValidateWorkload(workload); err == nil {
		t.Fatal("expected error for invalid workload state")
	}
}

func TestValidateWorkloadRequiresWorkerWhenPlaced(t *testing.T) {
	workload := Workload{
		ID:               "workload-1",
		ModelID:          "llama-3",
		Priority:         PriorityStandard,
		RequiredMemoryMB: 16000,
		State:            WorkloadPlaced,
	}

	if err := ValidateWorkload(workload); err == nil {
		t.Fatal(
			"expected placed workload without assigned worker to fail",
		)
	}
}

func TestValidateWorkloadRequiresWorkerWhenRunning(t *testing.T) {
	workload := Workload{
		ID:               "workload-1",
		ModelID:          "llama-3",
		Priority:         PriorityCritical,
		RequiredMemoryMB: 24000,
		State:            WorkloadRunning,
	}

	if err := ValidateWorkload(workload); err == nil {
		t.Fatal(
			"expected running workload without assigned worker to fail",
		)
	}
}

func TestValidateWorkloadAcceptsRunningWorkloadWithWorker(t *testing.T) {
	workload := Workload{
		ID:               "workload-1",
		ModelID:          "llama-3",
		Priority:         PriorityCritical,
		RequiredMemoryMB: 24000,
		State:            WorkloadRunning,
		AssignedWorkerID: "worker-7",
	}

	if err := ValidateWorkload(workload); err != nil {
		t.Fatalf(
			"expected running workload with assigned worker to be valid, got: %v",
			err,
		)
	}
}
