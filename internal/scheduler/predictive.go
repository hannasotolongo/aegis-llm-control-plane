package scheduler

import (
	"time"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
	"github.com/hannasotolongo/aegis-llm-control-plane/internal/predictor"
)

const (
	predictedMemoryPenaltyWeight  = 1.0
	predictedComputePenaltyWeight = 1.0
	predictedContentionPenalty    = 100.0
)

type PredictionProvider interface {
	Get(workerID string) (predictor.Result, bool)
}

type PredictiveScoreBreakdown struct {
	Base                       WorkerScoreBreakdown
	PredictedMemoryPenalty     float64
	PredictedComputePenalty    float64
	PredictedContentionPenalty float64
	UsedPrediction             bool
	Total                      float64
}

func SelectWorkerPredictive(
	workload cluster.Workload,
	workers []cluster.Worker,
	predictions PredictionProvider,
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

		score := ExplainPredictiveWorkerScore(
			workload,
			worker,
			predictions,
		).Total

		if !found || score > selectedScore {
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

func ExplainPredictiveWorkerScore(
	workload cluster.Workload,
	worker cluster.Worker,
	predictions PredictionProvider,
) PredictiveScoreBreakdown {
	base := ExplainWorkerScore(workload, worker)

	breakdown := PredictiveScoreBreakdown{
		Base:  base,
		Total: base.Total,
	}

	if predictions == nil {
		return breakdown
	}

	result, ok := predictions.Get(worker.ID)
	if !ok {
		return breakdown
	}

	if time.Since(result.GeneratedAt) > 15*time.Second {
		return breakdown
	}

	breakdown.UsedPrediction = true

	breakdown.PredictedMemoryPenalty =
		result.Forecast.PredictedMemoryUtilization *
			predictedMemoryPenaltyWeight

	breakdown.PredictedComputePenalty =
		result.Forecast.PredictedComputeUtilization *
			predictedComputePenaltyWeight

	if result.Forecast.PredictedContention {
		breakdown.PredictedContentionPenalty =
			predictedContentionPenalty
	}

	breakdown.Total =
		base.Total -
			breakdown.PredictedMemoryPenalty -
			breakdown.PredictedComputePenalty -
			breakdown.PredictedContentionPenalty

	return breakdown
}
