package cluster

import (
	"context"
	"fmt"
	"time"
)

type ReconcileResult struct {
	UpdatedWorkers     int
	RecoveredWorkloads int
	FailedWorkloads    int
}

func ReconcileCluster(
	ctx context.Context,
	store StateStore,
	now time.Time,
	config WorkerHealthConfig,
) (ReconcileResult, error) {
	snapshot, err := store.Snapshot(ctx)
	if err != nil {
		return ReconcileResult{}, err
	}

	result := ReconcileResult{}

	workerStates := make(map[string]WorkerState, len(snapshot.Workers))

	for _, worker := range snapshot.Workers {
		newState := EvaluateWorkerHealth(
			worker,
			now,
			config,
		)

		workerStates[worker.ID] = newState

		if newState == worker.State {
			continue
		}

		worker.State = newState

		if err := store.UpdateWorker(ctx, worker); err != nil {
			return result, fmt.Errorf(
				"update worker %q: %w",
				worker.ID,
				err,
			)
		}

		result.UpdatedWorkers++
	}

	for _, workload := range snapshot.Workloads {
		if workload.AssignedWorkerID == "" {
			continue
		}

		workerState, exists := workerStates[workload.AssignedWorkerID]
		if !exists {
			continue
		}

		if workerState != WorkerUnhealthy {
			continue
		}

		if workload.State != WorkloadPlaced &&
			workload.State != WorkloadRunning {
			continue
		}

		if workload.Checkpointable {
			_, err := store.ReleasePlacement(
				ctx,
				workload.ID,
				WorkloadRecovering,
			)
			if err != nil {
				return result, fmt.Errorf(
					"recover workload %q: %w",
					workload.ID,
					err,
				)
			}

			result.RecoveredWorkloads++
			continue
		}

		_, err := store.ReleasePlacement(
			ctx,
			workload.ID,
			WorkloadFailed,
		)
		if err != nil {
			return result, fmt.Errorf(
				"fail workload %q: %w",
				workload.ID,
				err,
			)
		}

		result.FailedWorkloads++
	}

	return result, nil
}
