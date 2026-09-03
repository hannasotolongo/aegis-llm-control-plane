package telemetry

import (
	"context"
	"time"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
)

type Collector struct {
	store Store
}

func NewCollector(store Store) *Collector {
	return &Collector{
		store: store,
	}
}

func (c *Collector) RecordSnapshot(
	ctx context.Context,
	snapshot cluster.Snapshot,
	timestamp time.Time,
) error {
	workloadsByWorker := make(map[string][]cluster.Workload)

	for _, workload := range snapshot.Workloads {
		if workload.AssignedWorkerID == "" {
			continue
		}

		workloadsByWorker[workload.AssignedWorkerID] = append(
			workloadsByWorker[workload.AssignedWorkerID],
			workload,
		)
	}

	for _, worker := range snapshot.Workers {
		workloads := workloadsByWorker[worker.ID]

		if len(workloads) == 0 {
			record := Record{
				Timestamp:           timestamp,
				WorkerID:            worker.ID,
				WorkerState:         worker.State,
				AvailableMemoryMB:   worker.AvailableMemoryMB,
				TotalMemoryMB:       worker.TotalMemoryMB,
				ComputeUtilization:  worker.ComputeUtilization,
				MemoryUtilization:   worker.MemoryUtilization,
				ActiveWorkloadCount: worker.ActiveWorkloadCount,
				TopologyDomain:      worker.TopologyDomain,
			}

			if err := c.store.Append(ctx, record); err != nil {
				return err
			}

			continue
		}

		for _, workload := range workloads {
			record := Record{
				Timestamp:           timestamp,
				WorkerID:            worker.ID,
				WorkerState:         worker.State,
				AvailableMemoryMB:   worker.AvailableMemoryMB,
				TotalMemoryMB:       worker.TotalMemoryMB,
				ComputeUtilization:  worker.ComputeUtilization,
				MemoryUtilization:   worker.MemoryUtilization,
				ActiveWorkloadCount: worker.ActiveWorkloadCount,
				TopologyDomain:      worker.TopologyDomain,
				WorkloadID:          workload.ID,
				WorkloadState:       workload.State,
				ModelID:             workload.ModelID,
				Priority:            workload.Priority,
				RequiredMemoryMB:    workload.RequiredMemoryMB,
				EstimatedCompute:    workload.EstimatedCompute,
				LatencySLO:          workload.LatencySLO,
				Checkpointable:      workload.Checkpointable,
				AssignedWorkerID:    workload.AssignedWorkerID,
			}

			if err := c.store.Append(ctx, record); err != nil {
				return err
			}

		}
	}

	return nil
}
