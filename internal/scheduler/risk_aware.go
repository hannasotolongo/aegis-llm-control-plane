package scheduler

import (
	"time"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
	"github.com/hannasotolongo/aegis-llm-control-plane/internal/risk"
)

type RiskAwareScoreBreakdown struct {
	Base          WorkerScoreBreakdown
	PlacementRisk risk.PlacementRisk
	UsedRisk      bool
}

func SelectWorkerRiskAware(
	workload cluster.Workload,
	workers []cluster.Worker,
	predictions PredictionProvider,
	evaluator *risk.Evaluator,
) (cluster.Worker, error) {
	var selected cluster.Worker
	var selectedBreakdown RiskAwareScoreBreakdown

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

		breakdown := ExplainRiskAwareWorkerScore(
			workload,
			worker,
			predictions,
			evaluator,
		)

		if !found || preferRiskAware(
			breakdown,
			selectedBreakdown,
		) {
			selected = worker
			selectedBreakdown = breakdown
			found = true
		}
	}

	if !found {
		return cluster.Worker{}, ErrNoEligibleWorker
	}

	return selected, nil
}

func ExplainRiskAwareWorkerScore(
	workload cluster.Workload,
	worker cluster.Worker,
	predictions PredictionProvider,
	evaluator *risk.Evaluator,
) RiskAwareScoreBreakdown {
	base := ExplainWorkerScore(
		workload,
		worker,
	)

	breakdown := RiskAwareScoreBreakdown{
		Base: base,
	}

	if predictions == nil || evaluator == nil {
		return breakdown
	}

	result, ok := predictions.Get(worker.ID)
	if !ok {
		return breakdown
	}

	if time.Since(result.GeneratedAt) >
		15*time.Second {
		return breakdown
	}

	breakdown.PlacementRisk =
		evaluator.Evaluate(
			risk.Inputs{
				Workload: workload,
				Worker:   worker,
				Forecast: result,
			},
		)

	breakdown.UsedRisk = true

	return breakdown
}

func preferRiskAware(
	candidate RiskAwareScoreBreakdown,
	current RiskAwareScoreBreakdown,
) bool {
	if candidate.UsedRisk &&
		current.UsedRisk {
		candidateLevel := riskLevelRank(
			candidate.PlacementRisk.Level,
		)

		currentLevel := riskLevelRank(
			current.PlacementRisk.Level,
		)

		if candidateLevel != currentLevel {
			return candidateLevel <
				currentLevel
		}

		if candidate.PlacementRisk.Score !=
			current.PlacementRisk.Score {
			return candidate.PlacementRisk.Score <
				current.PlacementRisk.Score
		}
	}

	return candidate.Base.Total >
		current.Base.Total
}

func riskLevelRank(
	level risk.Level,
) int {
	switch level {
	case risk.LevelLow:
		return 0

	case risk.LevelModerate:
		return 1

	case risk.LevelHigh:
		return 2

	case risk.LevelCritical:
		return 3

	default:
		return 4
	}
}
