package risk

import (
	"testing"
	"time"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
	"github.com/hannasotolongo/aegis-llm-control-plane/internal/predictor"
)

func TestEvaluateMemoryRisk(t *testing.T) {
	evaluator := NewEvaluator()

	inputs := Inputs{
		Workload: cluster.Workload{
			ID:               "workload-1",
			RequiredMemoryMB: 16000,
		},
		Worker: cluster.Worker{
			ID:            "worker-1",
			TotalMemoryMB: 80000,
		},
		Forecast: predictor.Result{
			GeneratedAt: time.Now(),
			Forecast: predictor.Forecast{
				PredictedMemoryUtilization: 55,
			},
		},
	}

	result := evaluator.Evaluate(inputs)

	if result.Score != 75 {
		t.Fatalf(
			"expected risk score 75, got %.2f",
			result.Score,
		)
	}

	if result.Level != LevelHigh {
		t.Fatalf(
			"expected HIGH risk level, got %s",
			result.Level,
		)
	}

	if result.Breakdown.MemoryPressure != 75 {
		t.Fatalf(
			"expected memory pressure 75, got %.2f",
			result.Breakdown.MemoryPressure,
		)
	}
}

func TestEvaluateMemoryRiskClampsAt100(t *testing.T) {
	evaluator := NewEvaluator()

	inputs := Inputs{
		Workload: cluster.Workload{
			ID:               "workload-1",
			RequiredMemoryMB: 40000,
		},
		Worker: cluster.Worker{
			ID:            "worker-1",
			TotalMemoryMB: 80000,
		},
		Forecast: predictor.Result{
			GeneratedAt: time.Now(),
			Forecast: predictor.Forecast{
				PredictedMemoryUtilization: 80,
			},
		},
	}

	result := evaluator.Evaluate(inputs)

	if result.Score != 100 {
		t.Fatalf(
			"expected clamped risk score 100, got %.2f",
			result.Score,
		)
	}

	if result.Level != LevelCritical {
		t.Fatalf(
			"expected CRITICAL risk level, got %s",
			result.Level,
		)
	}
}

func TestEvaluateZeroMemoryWorkerIsCritical(t *testing.T) {
	evaluator := NewEvaluator()

	inputs := Inputs{
		Workload: cluster.Workload{
			ID:               "workload-1",
			RequiredMemoryMB: 1000,
		},
		Worker: cluster.Worker{
			ID:            "worker-1",
			TotalMemoryMB: 0,
		},
	}

	result := evaluator.Evaluate(inputs)

	if result.Score != 100 {
		t.Fatalf(
			"expected risk score 100, got %.2f",
			result.Score,
		)
	}

	if result.Level != LevelCritical {
		t.Fatalf(
			"expected CRITICAL risk level, got %s",
			result.Level,
		)
	}
}
func TestEvaluateComputePressureCanDriveRisk(t *testing.T) {
	evaluator := NewEvaluator()

	inputs := Inputs{
		Workload: cluster.Workload{
			ID:               "workload-1",
			RequiredMemoryMB: 4000,
		},
		Worker: cluster.Worker{
			ID:            "worker-1",
			TotalMemoryMB: 80000,
		},
		Forecast: predictor.Result{
			GeneratedAt: time.Now(),
			Forecast: predictor.Forecast{
				PredictedMemoryUtilization:  20,
				PredictedComputeUtilization: 82,
			},
		},
	}

	result := evaluator.Evaluate(inputs)

	if result.Score != 82 {
		t.Fatalf(
			"expected risk score 82, got %.2f",
			result.Score,
		)
	}

	if result.Level != LevelHigh {
		t.Fatalf(
			"expected HIGH risk level, got %s",
			result.Level,
		)
	}

	if result.Breakdown.ComputePressure != 82 {
		t.Fatalf(
			"expected compute pressure 82, got %.2f",
			result.Breakdown.ComputePressure,
		)
	}
}
func TestHigherForecastUncertaintyIncreasesRisk(t *testing.T) {
	evaluator := NewEvaluator()

	baseInputs := Inputs{
		Workload: cluster.Workload{
			ID:               "workload-1",
			RequiredMemoryMB: 16000,
		},
		Worker: cluster.Worker{
			ID:            "worker-1",
			TotalMemoryMB: 80000,
		},
		Forecast: predictor.Result{
			GeneratedAt: time.Now(),
			Forecast: predictor.Forecast{
				PredictedMemoryUtilization:  50,
				PredictedComputeUtilization: 40,
			},
		},
	}

	lowUncertainty := baseInputs
	lowUncertainty.Forecast.Forecast.Uncertainty = predictor.Uncertainty{
		MemoryError:  2,
		ComputeError: 2,
		SampleCount:  20,
	}

	highUncertainty := baseInputs
	highUncertainty.Forecast.Forecast.Uncertainty = predictor.Uncertainty{
		MemoryError:  12,
		ComputeError: 12,
		SampleCount:  20,
	}

	lowRisk := evaluator.Evaluate(lowUncertainty)
	highRisk := evaluator.Evaluate(highUncertainty)

	if highRisk.Score <= lowRisk.Score {
		t.Fatalf(
			"expected higher uncertainty to increase risk: low %.2f, high %.2f",
			lowRisk.Score,
			highRisk.Score,
		)
	}

	if lowRisk.Breakdown.ForecastUncertainty != 2 {
		t.Fatalf(
			"expected low forecast uncertainty 2, got %.2f",
			lowRisk.Breakdown.ForecastUncertainty,
		)
	}

	if highRisk.Breakdown.ForecastUncertainty != 12 {
		t.Fatalf(
			"expected high forecast uncertainty 12, got %.2f",
			highRisk.Breakdown.ForecastUncertainty,
		)
	}
}
