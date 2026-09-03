package scheduler

import (
	"context"
	"fmt"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
)

func PlaceWorkloadPredictive(
	ctx context.Context,
	store cluster.StateStore,
	workloadID string,
	predictions PredictionProvider,
) (cluster.Workload, error) {
	placed, _, err := PlaceWorkloadPredictiveWithDecision(
		ctx,
		store,
		workloadID,
		predictions,
	)
	if err != nil {
		return cluster.Workload{}, err
	}

	return placed, nil
}

func PlaceWorkloadPredictiveWithDecision(
	ctx context.Context,
	store cluster.StateStore,
	workloadID string,
	predictions PredictionProvider,
) (cluster.Workload, PredictiveScoreBreakdown, error) {
	snapshot, err := store.Snapshot(ctx)
	if err != nil {
		return cluster.Workload{}, PredictiveScoreBreakdown{}, err
	}

	var workload cluster.Workload
	found := false

	for _, candidate := range snapshot.Workloads {
		if candidate.ID == workloadID {
			workload = candidate
			found = true
			break
		}
	}

	if !found {
		return cluster.Workload{}, PredictiveScoreBreakdown{}, fmt.Errorf(
			"%w: %s",
			cluster.ErrWorkloadNotFound,
			workloadID,
		)
	}

	if workload.State != cluster.WorkloadPending &&
		workload.State != cluster.WorkloadQueued {
		return cluster.Workload{}, PredictiveScoreBreakdown{}, fmt.Errorf(
			"workload %q is not schedulable from state %q",
			workload.ID,
			workload.State,
		)
	}

	selected, err := SelectWorkerPredictive(
		workload,
		snapshot.Workers,
		predictions,
	)
	if err != nil {
		return cluster.Workload{}, PredictiveScoreBreakdown{}, err
	}

	score := ExplainPredictiveWorkerScore(
		workload,
		selected,
		predictions,
	)

	placed, err := store.CommitPlacement(
		ctx,
		workload.ID,
		selected.ID,
	)
	if err != nil {
		return cluster.Workload{}, PredictiveScoreBreakdown{}, err
	}

	return placed, score, nil
}
