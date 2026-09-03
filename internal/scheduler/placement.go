package scheduler

import (
	"context"
	"fmt"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
)

type PlacementDecision struct {
	WorkloadID string
	WorkerID   string
	Score      WorkerScoreBreakdown
	Reasons    []string
}

func PlaceWorkload(
	ctx context.Context,
	store cluster.StateStore,
	workloadID string,
) (cluster.Workload, error) {
	placed, _, err := PlaceWorkloadWithDecision(
		ctx,
		store,
		workloadID,
	)
	if err != nil {
		return cluster.Workload{}, err
	}

	return placed, nil
}

func PlaceWorkloadWithDecision(
	ctx context.Context,
	store cluster.StateStore,
	workloadID string,
) (cluster.Workload, PlacementDecision, error) {
	snapshot, err := store.Snapshot(ctx)
	if err != nil {
		return cluster.Workload{}, PlacementDecision{}, err
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
		return cluster.Workload{}, PlacementDecision{}, fmt.Errorf(
			"%w: %s",
			cluster.ErrWorkloadNotFound,
			workloadID,
		)
	}

	if workload.State != cluster.WorkloadPending &&
		workload.State != cluster.WorkloadQueued {
		return cluster.Workload{}, PlacementDecision{}, fmt.Errorf(
			"workload %q is not schedulable from state %q",
			workload.ID,
			workload.State,
		)
	}

	selected, err := SelectWorker(workload, snapshot.Workers)
	if err != nil {
		return cluster.Workload{}, PlacementDecision{}, err
	}

	score := ExplainWorkerScore(workload, selected)
	reasons := buildPlacementReasons(score)

	placed, err := store.CommitPlacement(
		ctx,
		workload.ID,
		selected.ID,
	)
	if err != nil {
		return cluster.Workload{}, PlacementDecision{}, err
	}

	decision := PlacementDecision{
		WorkloadID: workload.ID,
		WorkerID:   selected.ID,
		Score:      score,
		Reasons:    reasons,
	}

	return placed, decision, nil
}

func buildPlacementReasons(
	score WorkerScoreBreakdown,
) []string {
	reasons := make([]string, 0, 4)

	if score.ModelLocalityBonus > 0 {
		reasons = append(
			reasons,
			"requested model is already cached on the worker",
		)
	}

	if score.ComputePenalty < 50 {
		reasons = append(
			reasons,
			"worker has relatively low compute utilization",
		)
	}

	if score.MemoryPenalty < 50 {
		reasons = append(
			reasons,
			"worker has relatively low memory utilization",
		)
	}

	if score.ActiveWorkloadPenalty <= 10 {
		reasons = append(
			reasons,
			"worker has a low active workload count",
		)
	}

	if len(reasons) == 0 {
		reasons = append(
			reasons,
			"worker had the highest eligible scheduling score",
		)
	}

	return reasons
}
