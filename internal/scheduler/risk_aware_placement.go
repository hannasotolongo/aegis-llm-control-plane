package scheduler

import (
	"context"
	"fmt"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
	"github.com/hannasotolongo/aegis-llm-control-plane/internal/risk"
)

func PlaceWorkloadRiskAware(
	ctx context.Context,
	store cluster.StateStore,
	workloadID string,
	predictions PredictionProvider,
	evaluator *risk.Evaluator,
) (cluster.Workload, error) {
	placed, _, err := PlaceWorkloadRiskAwareWithDecision(
		ctx,
		store,
		workloadID,
		predictions,
		evaluator,
	)
	if err != nil {
		return cluster.Workload{}, err
	}

	return placed, nil
}

func PlaceWorkloadRiskAwareWithDecision(
	ctx context.Context,
	store cluster.StateStore,
	workloadID string,
	predictions PredictionProvider,
	evaluator *risk.Evaluator,
) (
	cluster.Workload,
	RiskAwareScoreBreakdown,
	error,
) {
	snapshot, err := store.Snapshot(ctx)
	if err != nil {
		return cluster.Workload{},
			RiskAwareScoreBreakdown{},
			err
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
		return cluster.Workload{},
			RiskAwareScoreBreakdown{},
			fmt.Errorf(
				"%w: %s",
				cluster.ErrWorkloadNotFound,
				workloadID,
			)
	}

	if workload.State != cluster.WorkloadPending &&
		workload.State != cluster.WorkloadQueued {
		return cluster.Workload{},
			RiskAwareScoreBreakdown{},
			fmt.Errorf(
				"workload %q is not schedulable from state %q",
				workload.ID,
				workload.State,
			)
	}

	selected, err := SelectWorkerRiskAware(
		workload,
		snapshot.Workers,
		predictions,
		evaluator,
	)
	if err != nil {
		return cluster.Workload{},
			RiskAwareScoreBreakdown{},
			err
	}

	score := ExplainRiskAwareWorkerScore(
		workload,
		selected,
		predictions,
		evaluator,
	)

	placed, err := store.CommitPlacement(
		ctx,
		workload.ID,
		selected.ID,
	)
	if err != nil {
		return cluster.Workload{},
			RiskAwareScoreBreakdown{},
			err
	}

	return placed, score, nil
}
