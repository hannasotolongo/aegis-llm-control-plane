package main

import (
	"fmt"
	"math"
	"time"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
	"github.com/hannasotolongo/aegis-llm-control-plane/internal/predictor"
	"github.com/hannasotolongo/aegis-llm-control-plane/internal/risk"
	"github.com/hannasotolongo/aegis-llm-control-plane/internal/scheduler"
)

const contentionThreshold = 85.0

type benchmarkPredictionProvider struct {
	results map[string]predictor.Result
}

func (p *benchmarkPredictionProvider) Get(
	workerID string,
) (predictor.Result, bool) {
	result, ok := p.results[workerID]
	return result, ok
}

type scenario struct {
	Name        string
	Predictions func([]cluster.Worker) *benchmarkPredictionProvider
	Diagnostic  bool
}

type benchmarkResult struct {
	Name string

	Placements    int
	Rejections    int
	WorkerChoices map[string]int

	TotalLatency time.Duration

	PeakMemoryUtil  float64
	PeakComputeUtil float64

	ContentionTransitions int
	PressureSteps         int

	TotalClusterMemoryPressure  float64
	TotalClusterComputePressure float64
	PressureSamples             int

	FinalLoadImbalance int
}

func main() {
	fmt.Println("Aegis Multi-Scenario Scheduler Benchmark")
	fmt.Println("========================================")
	fmt.Printf(
		"Contention threshold: %.0f%%\n\n",
		contentionThreshold,
	)

	scenarios := []scenario{
		{
			Name:        "Steady Forecast",
			Predictions: steadyPredictions,
		},
		{
			Name:        "Rising Forecast Pressure",
			Predictions: risingContentionPredictions,
		},
		{
			Name:        "Forecast Uncertainty",
			Predictions: uncertaintyPredictions,
		},
		{
			Name:        "Stale Forecast",
			Predictions: stalePredictions,
		},
	}

	for _, s := range scenarios {
		runScenario(s)
	}
}

func runScenario(s scenario) {
	fmt.Printf("SCENARIO: %s\n", s.Name)
	fmt.Println("------------------------------")

	workloads := benchmarkWorkloads(48)

	baseline := runBaselineBenchmark(workloads)

	predictive := runPredictiveBenchmark(
		workloads,
		s.Predictions,
	)

	riskAware := runRiskAwareBenchmark(
		workloads,
		s.Predictions,
		s.Diagnostic,
	)

	printResult(baseline)
	printResult(predictive)
	printResult(riskAware)

	fmt.Println()
}

func runBaselineBenchmark(
	workloads []cluster.Workload,
) benchmarkResult {
	workers := benchmarkWorkers()
	result := newBenchmarkResult("Baseline")

	previousPressure := make(map[string]bool)
	active := make([]benchmarkActiveWorkload, 0)

	for step, workload := range workloads {
		releaseCompletedWorkloads(
			&workers,
			&active,
			step,
		)

		start := time.Now()

		selected, err := scheduler.SelectWorker(
			workload,
			workers,
		)

		result.TotalLatency += time.Since(start)

		if err != nil {
			result.Rejections++
			continue
		}

		result.Placements++
		result.WorkerChoices[selected.ID]++

		applyPlacement(
			&workers,
			selected.ID,
			workload,
		)

		trackBenchmarkWorkload(
			&active,
			workload,
			selected.ID,
			step,
		)

		recordPressure(
			&result,
			workers,
			previousPressure,
		)
	}

	result.FinalLoadImbalance =
		calculateLoadImbalance(workers)

	return result
}

func runPredictiveBenchmark(
	workloads []cluster.Workload,
	predictionFn func(
		[]cluster.Worker,
	) *benchmarkPredictionProvider,
) benchmarkResult {
	workers := benchmarkWorkers()
	result := newBenchmarkResult("Predictive")

	previousPressure := make(map[string]bool)
	active := make([]benchmarkActiveWorkload, 0)

	for step, workload := range workloads {
		releaseCompletedWorkloads(
			&workers,
			&active,
			step,
		)

		predictions :=
			predictionFn(workers)

		start := time.Now()

		selected, err :=
			scheduler.SelectWorkerPredictive(
				workload,
				workers,
				predictions,
			)

		result.TotalLatency += time.Since(start)

		if err != nil {
			result.Rejections++
			continue
		}

		result.Placements++
		result.WorkerChoices[selected.ID]++

		applyPlacement(
			&workers,
			selected.ID,
			workload,
		)

		trackBenchmarkWorkload(
			&active,
			workload,
			selected.ID,
			step,
		)

		recordPressure(
			&result,
			workers,
			previousPressure,
		)
	}

	result.FinalLoadImbalance =
		calculateLoadImbalance(workers)

	return result
}

func runRiskAwareBenchmark(
	workloads []cluster.Workload,
	predictionFn func(
		[]cluster.Worker,
	) *benchmarkPredictionProvider,
	diagnostic bool,
) benchmarkResult {
	workers := benchmarkWorkers()
	result := newBenchmarkResult("Risk-Aware")

	previousPressure := make(map[string]bool)
	active := make([]benchmarkActiveWorkload, 0)
	evaluator := risk.NewEvaluator()

	for decision, workload := range workloads {
		releaseCompletedWorkloads(
			&workers,
			&active,
			decision,
		)

		predictions :=
			predictionFn(workers)

		if diagnostic {
			printRiskDiagnostic(
				decision+1,
				workload,
				workers,
				predictions,
				evaluator,
			)
		}

		start := time.Now()

		selected, err :=
			scheduler.SelectWorkerRiskAware(
				workload,
				workers,
				predictions,
				evaluator,
			)

		result.TotalLatency += time.Since(start)

		if err != nil {
			result.Rejections++

			if diagnostic {
				fmt.Println(
					"  Selected: none",
				)
				fmt.Println()
			}

			continue
		}

		if diagnostic {
			fmt.Printf(
				"  Selected: %s\n\n",
				selected.ID,
			)
		}

		result.Placements++
		result.WorkerChoices[selected.ID]++

		applyPlacement(
			&workers,
			selected.ID,
			workload,
		)

		trackBenchmarkWorkload(
			&active,
			workload,
			selected.ID,
			decision,
		)

		recordPressure(
			&result,
			workers,
			previousPressure,
		)
	}

	result.FinalLoadImbalance =
		calculateLoadImbalance(workers)

	return result
}

func printRiskDiagnostic(
	decision int,
	workload cluster.Workload,
	workers []cluster.Worker,
	predictions *benchmarkPredictionProvider,
	evaluator *risk.Evaluator,
) {
	fmt.Printf(
		"Risk decision %d — %s\n",
		decision,
		workload.ID,
	)

	for _, worker := range workers {
		if worker.State != cluster.WorkerHealthy {
			continue
		}

		if worker.AvailableMemoryMB <
			workload.RequiredMemoryMB {
			continue
		}

		breakdown :=
			scheduler.ExplainRiskAwareWorkerScore(
				workload,
				worker,
				predictions,
				evaluator,
			)

		if !breakdown.UsedRisk {
			fmt.Printf(
				"  %s: baseline=%.2f risk=unavailable\n",
				worker.ID,
				breakdown.Base.Total,
			)
			continue
		}

		fmt.Printf(
			"  %s: baseline=%.2f risk=%s %.2f memory=%.1f%% compute=%.1f%%\n",
			worker.ID,
			breakdown.Base.Total,
			breakdown.PlacementRisk.Level,
			breakdown.PlacementRisk.Score,
			worker.MemoryUtilization,
			worker.ComputeUtilization,
		)
	}
}

func applyPlacement(
	workers *[]cluster.Worker,
	workerID string,
	workload cluster.Workload,
) {
	for i := range *workers {
		worker := &(*workers)[i]

		if worker.ID != workerID {
			continue
		}

		worker.AvailableMemoryMB -=
			workload.RequiredMemoryMB

		if worker.TotalMemoryMB > 0 {
			usedMemory :=
				worker.TotalMemoryMB -
					worker.AvailableMemoryMB

			worker.MemoryUtilization =
				float64(usedMemory) /
					float64(worker.TotalMemoryMB) *
					100
		}

		worker.ActiveWorkloadCount++

		worker.ComputeUtilization += 12

		if worker.ComputeUtilization > 100 {
			worker.ComputeUtilization = 100
		}

		return
	}
}

func recordPressure(
	result *benchmarkResult,
	workers []cluster.Worker,
	previousPressure map[string]bool,
) {
	var clusterMemory float64
	var clusterCompute float64

	stepUnderPressure := false

	for _, worker := range workers {
		if worker.MemoryUtilization >
			result.PeakMemoryUtil {
			result.PeakMemoryUtil =
				worker.MemoryUtilization
		}

		if worker.ComputeUtilization >
			result.PeakComputeUtil {
			result.PeakComputeUtil =
				worker.ComputeUtilization
		}

		clusterMemory +=
			worker.MemoryUtilization

		clusterCompute +=
			worker.ComputeUtilization

		currentlyUnderPressure :=
			worker.MemoryUtilization >=
				contentionThreshold ||
				worker.ComputeUtilization >=
					contentionThreshold

		if currentlyUnderPressure {
			stepUnderPressure = true
		}

		if currentlyUnderPressure &&
			!previousPressure[worker.ID] {
			result.ContentionTransitions++
		}

		previousPressure[worker.ID] =
			currentlyUnderPressure
	}

	if stepUnderPressure {
		result.PressureSteps++
	}

	if len(workers) > 0 {
		result.TotalClusterMemoryPressure +=
			clusterMemory /
				float64(len(workers))

		result.TotalClusterComputePressure +=
			clusterCompute /
				float64(len(workers))

		result.PressureSamples++
	}
}

func calculateLoadImbalance(
	workers []cluster.Worker,
) int {
	if len(workers) == 0 {
		return 0
	}

	minimum :=
		workers[0].ActiveWorkloadCount

	maximum :=
		workers[0].ActiveWorkloadCount

	for _, worker := range workers[1:] {
		if worker.ActiveWorkloadCount <
			minimum {
			minimum =
				worker.ActiveWorkloadCount
		}

		if worker.ActiveWorkloadCount >
			maximum {
			maximum =
				worker.ActiveWorkloadCount
		}
	}

	return int(math.Abs(
		float64(maximum - minimum),
	))
}

func steadyPredictions(
	workers []cluster.Worker,
) *benchmarkPredictionProvider {
	results := make(
		map[string]predictor.Result,
		len(workers),
	)

	now := time.Now()

	for _, worker := range workers {
		results[worker.ID] =
			predictor.Result{
				GeneratedAt: now,
				Forecast: predictor.Forecast{
					PredictedMemoryUtilization: clamp(
						worker.MemoryUtilization + 8,
					),
					PredictedComputeUtilization: clamp(
						worker.ComputeUtilization + 8,
					),
					Uncertainty: predictor.Uncertainty{
						MemoryError:  3,
						ComputeError: 3,
						SampleCount:  20,
					},
				},
			}
	}

	return &benchmarkPredictionProvider{
		results: results,
	}
}

func risingContentionPredictions(
	workers []cluster.Worker,
) *benchmarkPredictionProvider {
	results := make(
		map[string]predictor.Result,
		len(workers),
	)

	now := time.Now()

	for _, worker := range workers {
		memoryGrowth := 8.0
		computeGrowth := 8.0
		memoryError := 4.0
		computeError := 4.0

		if worker.ID == "worker-1" {
			memoryGrowth = 22
			computeGrowth = 25
			memoryError = 6
			computeError = 6
		}

		results[worker.ID] =
			predictor.Result{
				GeneratedAt: now,
				Forecast: predictor.Forecast{
					PredictedMemoryUtilization: clamp(
						worker.MemoryUtilization +
							memoryGrowth,
					),
					PredictedComputeUtilization: clamp(
						worker.ComputeUtilization +
							computeGrowth,
					),
					Uncertainty: predictor.Uncertainty{
						MemoryError:  memoryError,
						ComputeError: computeError,
						SampleCount:  20,
					},
				},
			}
	}

	return &benchmarkPredictionProvider{
		results: results,
	}
}

func uncertaintyPredictions(
	workers []cluster.Worker,
) *benchmarkPredictionProvider {
	results := make(
		map[string]predictor.Result,
		len(workers),
	)

	now := time.Now()

	for _, worker := range workers {
		predictedMemory :=
			worker.MemoryUtilization + 15

		predictedCompute :=
			worker.ComputeUtilization + 15

		memoryError := 3.0
		computeError := 3.0

		if worker.ID == "worker-1" {
			predictedMemory =
				worker.MemoryUtilization + 5

			predictedCompute =
				worker.ComputeUtilization + 5

			memoryError = 25
			computeError = 25
		}

		results[worker.ID] =
			predictor.Result{
				GeneratedAt: now,
				Forecast: predictor.Forecast{
					PredictedMemoryUtilization: clamp(
						predictedMemory,
					),
					PredictedComputeUtilization: clamp(
						predictedCompute,
					),
					Uncertainty: predictor.Uncertainty{
						MemoryError:  memoryError,
						ComputeError: computeError,
						SampleCount:  20,
					},
				},
			}
	}

	return &benchmarkPredictionProvider{
		results: results,
	}
}

func stalePredictions(
	workers []cluster.Worker,
) *benchmarkPredictionProvider {
	results := make(
		map[string]predictor.Result,
		len(workers),
	)

	// Deliberately make these predictions older than the
	// scheduler's 15-second freshness window.
	generatedAt := time.Now().Add(
		-30 * time.Second,
	)

	for _, worker := range workers {
		results[worker.ID] =
			predictor.Result{
				GeneratedAt: generatedAt,
				Forecast: predictor.Forecast{
					// These values are intentionally extreme.
					// A correct scheduler must ignore them because
					// the prediction is stale.
					PredictedMemoryUtilization:  100,
					PredictedComputeUtilization: 100,
					PredictedContention:         true,
					Uncertainty: predictor.Uncertainty{
						MemoryError:  30,
						ComputeError: 30,
						SampleCount:  20,
					},
				},
			}
	}

	return &benchmarkPredictionProvider{
		results: results,
	}
}

func benchmarkWorkers() []cluster.Worker {
	return []cluster.Worker{
		{
			ID:                  "worker-1",
			State:               cluster.WorkerHealthy,
			TotalMemoryMB:       80000,
			AvailableMemoryMB:   64000,
			MemoryUtilization:   20,
			ComputeUtilization:  20,
			ActiveWorkloadCount: 0,
			CachedModels: []string{
				"llama-3-8b",
			},
		},
		{
			ID:                  "worker-2",
			State:               cluster.WorkerHealthy,
			TotalMemoryMB:       80000,
			AvailableMemoryMB:   64000,
			MemoryUtilization:   20,
			ComputeUtilization:  20,
			ActiveWorkloadCount: 0,
		},
	}
}

func benchmarkWorkloads(
	count int,
) []cluster.Workload {
	workloads := make(
		[]cluster.Workload,
		0,
		count,
	)

	for i := 0; i < count; i++ {
		memory := uint64(8000)

		if i%3 == 0 {
			memory = 16000
		}

		workloads = append(
			workloads,
			cluster.Workload{
				ID: fmt.Sprintf(
					"benchmark-workload-%d",
					i,
				),
				ModelID:          "llama-3-8b",
				RequiredMemoryMB: memory,
			},
		)
	}

	return workloads
}

func newBenchmarkResult(
	name string,
) benchmarkResult {
	return benchmarkResult{
		Name: name,
		WorkerChoices: make(
			map[string]int,
		),
	}
}

func printResult(
	result benchmarkResult,
) {
	totalDecisions :=
		result.Placements +
			result.Rejections

	var averageLatency time.Duration
	var averageMemory float64
	var averageCompute float64

	if totalDecisions > 0 {
		averageLatency =
			result.TotalLatency /
				time.Duration(totalDecisions)
	}

	if result.PressureSamples > 0 {
		averageMemory =
			result.TotalClusterMemoryPressure /
				float64(result.PressureSamples)

		averageCompute =
			result.TotalClusterComputePressure /
				float64(result.PressureSamples)
	}

	fmt.Printf("%s\n", result.Name)

	fmt.Printf(
		"  Placements:             %d\n",
		result.Placements,
	)

	fmt.Printf(
		"  Rejections:             %d\n",
		result.Rejections,
	)

	fmt.Printf(
		"  Worker choices:         %v\n",
		result.WorkerChoices,
	)

	fmt.Printf(
		"  Peak memory:            %.1f%%\n",
		result.PeakMemoryUtil,
	)

	fmt.Printf(
		"  Peak compute:           %.1f%%\n",
		result.PeakComputeUtil,
	)

	fmt.Printf(
		"  Avg cluster memory:     %.1f%%\n",
		averageMemory,
	)

	fmt.Printf(
		"  Avg cluster compute:    %.1f%%\n",
		averageCompute,
	)

	fmt.Printf(
		"  Contention transitions: %d\n",
		result.ContentionTransitions,
	)

	fmt.Printf(
		"  Steps under pressure:   %d\n",
		result.PressureSteps,
	)

	fmt.Printf(
		"  Final active load imbalance: %d workloads\n",
		result.FinalLoadImbalance,
	)

	fmt.Printf(
		"  Average latency:        %s\n\n",
		averageLatency,
	)
}

func clamp(
	value float64,
) float64 {
	if value < 0 {
		return 0
	}

	if value > 100 {
		return 100
	}

	return value
}
