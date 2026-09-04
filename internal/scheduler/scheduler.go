package scheduler

import (
	"errors"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
)

var ErrNoEligibleWorker = errors.New("no eligible worker")

type WorkerScoreBreakdown struct {
	MemoryHeadroom        float64
	ComputePenalty        float64
	MemoryPenalty         float64
	ActiveWorkloadPenalty float64
	ModelLocalityBonus    float64
	Total                 float64
}

func SelectWorker(
	workload cluster.Workload,
	workers []cluster.Worker,
) (cluster.Worker, error) {
	var selected cluster.Worker
	var selectedScore float64
	found := false

	for _, worker := range workers {
		if worker.State != cluster.WorkerHealthy {
			continue
		}

		if worker.AvailableMemoryMB < workload.RequiredMemoryMB {
			continue
		}

		if !matchesTopology(workload, worker) {
			continue
		}

		score := scoreWorker(workload, worker)

		if !found ||
			score > selectedScore ||
			(score == selectedScore && worker.ID < selected.ID) {
			selected = worker
			selectedScore = score
			found = true
		}
	}

	if !found {
		return cluster.Worker{}, ErrNoEligibleWorker
	}

	return selected, nil
}

func ExplainWorkerScore(
	workload cluster.Workload,
	worker cluster.Worker,
) WorkerScoreBreakdown {
	breakdown := WorkerScoreBreakdown{
		MemoryHeadroom:        float64(worker.AvailableMemoryMB) / 1024,
		ComputePenalty:        worker.ComputeUtilization,
		MemoryPenalty:         worker.MemoryUtilization,
		ActiveWorkloadPenalty: float64(worker.ActiveWorkloadCount) * 10,
	}

	for _, modelID := range worker.CachedModels {
		if modelID == workload.ModelID {
			breakdown.ModelLocalityBonus = 100
			break
		}
	}

	breakdown.Total =
		breakdown.MemoryHeadroom -
			breakdown.ComputePenalty -
			breakdown.MemoryPenalty -
			breakdown.ActiveWorkloadPenalty +
			breakdown.ModelLocalityBonus

	return breakdown
}

func scoreWorker(
	workload cluster.Workload,
	worker cluster.Worker,
) float64 {
	return ExplainWorkerScore(workload, worker).Total
}
