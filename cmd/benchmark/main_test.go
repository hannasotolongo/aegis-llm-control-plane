package main

import (
	"reflect"
	"testing"
)

func TestStalePredictionsFallBackToBaseline(t *testing.T) {
	workloads := benchmarkWorkloads(48)

	baseline :=
		runBaselineBenchmark(workloads)

	predictive :=
		runPredictiveBenchmark(
			workloads,
			stalePredictions,
		)

	riskAware :=
		runRiskAwareBenchmark(
			workloads,
			stalePredictions,
			false,
		)

	if baseline.Placements != predictive.Placements {
		t.Fatalf(
			"predictive placements differ from baseline: baseline=%d predictive=%d",
			baseline.Placements,
			predictive.Placements,
		)
	}

	if baseline.Placements != riskAware.Placements {
		t.Fatalf(
			"risk-aware placements differ from baseline: baseline=%d risk-aware=%d",
			baseline.Placements,
			riskAware.Placements,
		)
	}

	if baseline.Rejections != predictive.Rejections {
		t.Fatalf(
			"predictive rejections differ from baseline: baseline=%d predictive=%d",
			baseline.Rejections,
			predictive.Rejections,
		)
	}

	if baseline.Rejections != riskAware.Rejections {
		t.Fatalf(
			"risk-aware rejections differ from baseline: baseline=%d risk-aware=%d",
			baseline.Rejections,
			riskAware.Rejections,
		)
	}

	if !reflect.DeepEqual(
		baseline.WorkerChoices,
		predictive.WorkerChoices,
	) {
		t.Fatalf(
			"predictive worker choices differ from baseline: baseline=%v predictive=%v",
			baseline.WorkerChoices,
			predictive.WorkerChoices,
		)
	}

	if !reflect.DeepEqual(
		baseline.WorkerChoices,
		riskAware.WorkerChoices,
	) {
		t.Fatalf(
			"risk-aware worker choices differ from baseline: baseline=%v risk-aware=%v",
			baseline.WorkerChoices,
			riskAware.WorkerChoices,
		)
	}

	if baseline.PressureSteps !=
		predictive.PressureSteps {
		t.Fatalf(
			"predictive pressure steps differ from baseline: baseline=%d predictive=%d",
			baseline.PressureSteps,
			predictive.PressureSteps,
		)
	}

	if baseline.PressureSteps !=
		riskAware.PressureSteps {
		t.Fatalf(
			"risk-aware pressure steps differ from baseline: baseline=%d risk-aware=%d",
			baseline.PressureSteps,
			riskAware.PressureSteps,
		)
	}
}
