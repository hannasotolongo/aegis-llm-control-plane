package cluster

import "fmt"

func ValidateWorker(worker Worker) error {
	if worker.ID == "" {
		return fmt.Errorf("worker ID cannot be empty")
	}

	if worker.NodeID == "" {
		return fmt.Errorf("worker %q must have a node ID", worker.ID)
	}

	if worker.TotalMemoryMB == 0 {
		return fmt.Errorf("worker %q must have GPU memory", worker.ID)
	}

	if worker.AvailableMemoryMB > worker.TotalMemoryMB {
		return fmt.Errorf(
			"worker %q available memory (%d MB) exceeds total memory (%d MB)",
			worker.ID,
			worker.AvailableMemoryMB,
			worker.TotalMemoryMB,
		)
	}

	if worker.ComputeUtilization < 0 || worker.ComputeUtilization > 100 {
		return fmt.Errorf(
			"worker %q compute utilization must be between 0 and 100",
			worker.ID,
		)
	}

	if worker.MemoryUtilization < 0 || worker.MemoryUtilization > 100 {
		return fmt.Errorf(
			"worker %q memory utilization must be between 0 and 100",
			worker.ID,
		)
	}

	return nil
}

func ValidateWorkload(workload Workload) error {
	if workload.ID == "" {
		return fmt.Errorf("workload ID cannot be empty")
	}

	if workload.ModelID == "" {
		return fmt.Errorf(
			"workload %q must have a model ID",
			workload.ID,
		)
	}

	if workload.RequiredMemoryMB == 0 {
		return fmt.Errorf(
			"workload %q must require GPU memory",
			workload.ID,
		)
	}

	if workload.EstimatedCompute < 0 {
		return fmt.Errorf(
			"workload %q estimated compute cannot be negative",
			workload.ID,
		)
	}

	if workload.ExpectedDuration < 0 {
		return fmt.Errorf(
			"workload %q expected duration cannot be negative",
			workload.ID,
		)
	}

	if workload.LatencySLO < 0 {
		return fmt.Errorf(
			"workload %q latency SLO cannot be negative",
			workload.ID,
		)
	}

	switch workload.Priority {
	case PriorityCritical, PriorityStandard, PriorityBatch:
	default:
		return fmt.Errorf(
			"workload %q has invalid priority %q",
			workload.ID,
			workload.Priority,
		)
	}

	switch workload.State {
	case WorkloadPending,
		WorkloadQueued,
		WorkloadPlaced,
		WorkloadRunning,
		WorkloadPreempted,
		WorkloadRecovering,
		WorkloadCompleted,
		WorkloadFailed:
	default:
		return fmt.Errorf(
			"workload %q has invalid state %q",
			workload.ID,
			workload.State,
		)
	}

	if workload.State == WorkloadPlaced ||
		workload.State == WorkloadRunning {
		if workload.AssignedWorkerID == "" {
			return fmt.Errorf(
				"workload %q in state %q must have an assigned worker",
				workload.ID,
				workload.State,
			)
		}
	}

	return nil
}
