package risk

import (
	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
	"github.com/hannasotolongo/aegis-llm-control-plane/internal/predictor"
)

type Inputs struct {
	Workload cluster.Workload
	Worker   cluster.Worker
	Forecast predictor.Result
}

type Evaluator struct{}

func NewEvaluator() *Evaluator {
	return &Evaluator{}
}

func (e *Evaluator) Evaluate(inputs Inputs) PlacementRisk {
	memoryPressure := calculateMemoryPressure(
		inputs.Workload,
		inputs.Worker,
		inputs.Forecast,
	)

	computePressure := calculateComputePressure(
		inputs.Forecast,
	)

	forecastUncertainty := calculateForecastUncertainty(
		inputs.Forecast,
	)

	score := combinePressure(
		memoryPressure,
		computePressure,
	)

	return PlacementRisk{
		WorkerID:   inputs.Worker.ID,
		WorkloadID: inputs.Workload.ID,
		Score:      score,
		Level:      levelForScore(score),
		Breakdown: Breakdown{
			MemoryPressure:      memoryPressure,
			ComputePressure:     computePressure,
			ForecastUncertainty: forecastUncertainty,
		},
	}
}

func calculateMemoryPressure(
	workload cluster.Workload,
	worker cluster.Worker,
	forecast predictor.Result,
) float64 {
	if worker.TotalMemoryMB == 0 {
		return 100
	}

	workloadMemoryPercent :=
		float64(workload.RequiredMemoryMB) /
			float64(worker.TotalMemoryMB) *
			100

	projectedMemory :=
		forecast.Forecast.PredictedMemoryUtilization +
			forecast.Forecast.Uncertainty.MemoryError +
			workloadMemoryPercent

	return clamp(projectedMemory, 0, 100)
}

func calculateComputePressure(
	forecast predictor.Result,
) float64 {
	projectedCompute :=
		forecast.Forecast.PredictedComputeUtilization +
			forecast.Forecast.Uncertainty.ComputeError

	return clamp(projectedCompute, 0, 100)
}

func calculateForecastUncertainty(
	forecast predictor.Result,
) float64 {
	memoryError := forecast.Forecast.Uncertainty.MemoryError
	computeError := forecast.Forecast.Uncertainty.ComputeError

	if memoryError > computeError {
		return clamp(memoryError, 0, 100)
	}

	return clamp(computeError, 0, 100)
}

func combinePressure(
	memoryPressure float64,
	computePressure float64,
) float64 {
	if memoryPressure > computePressure {
		return memoryPressure
	}

	return computePressure
}

func levelForScore(score float64) Level {
	switch {
	case score >= 85:
		return LevelCritical
	case score >= 70:
		return LevelHigh
	case score >= 50:
		return LevelModerate
	default:
		return LevelLow
	}
}

func clamp(value, minimum, maximum float64) float64 {
	if value < minimum {
		return minimum
	}

	if value > maximum {
		return maximum
	}

	return value
}
