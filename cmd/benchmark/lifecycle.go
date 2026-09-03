package main

import (
	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
)

type benchmarkActiveWorkload struct {
	Workload       cluster.Workload
	WorkerID       string
	CompleteAtStep int
}

func trackBenchmarkWorkload(
	active *[]benchmarkActiveWorkload,
	workload cluster.Workload,
	workerID string,
	currentStep int,
) {
	*active = append(
		*active,
		benchmarkActiveWorkload{
			Workload:       workload,
			WorkerID:       workerID,
			CompleteAtStep: currentStep + benchmarkDurationSteps(workload),
		},
	)
}

func releaseCompletedWorkloads(
	workers *[]cluster.Worker,
	active *[]benchmarkActiveWorkload,
	currentStep int,
) int {
	remaining := make(
		[]benchmarkActiveWorkload,
		0,
		len(*active),
	)

	completed := 0

	for _, running := range *active {
		if running.CompleteAtStep > currentStep {
			remaining = append(
				remaining,
				running,
			)
			continue
		}

		releaseBenchmarkPlacement(
			workers,
			running.WorkerID,
			running.Workload,
		)

		completed++
	}

	*active = remaining

	return completed
}

func releaseBenchmarkPlacement(
	workers *[]cluster.Worker,
	workerID string,
	workload cluster.Workload,
) {
	for i := range *workers {
		worker := &(*workers)[i]

		if worker.ID != workerID {
			continue
		}

		worker.AvailableMemoryMB +=
			workload.RequiredMemoryMB

		if worker.AvailableMemoryMB >
			worker.TotalMemoryMB {
			worker.AvailableMemoryMB =
				worker.TotalMemoryMB
		}

		if worker.TotalMemoryMB > 0 {
			usedMemory :=
				worker.TotalMemoryMB -
					worker.AvailableMemoryMB

			worker.MemoryUtilization =
				float64(usedMemory) /
					float64(worker.TotalMemoryMB) *
					100
		}

		if worker.ActiveWorkloadCount > 0 {
			worker.ActiveWorkloadCount--
		}

		if worker.ComputeUtilization >= 12 {
			worker.ComputeUtilization -= 12
		} else {
			worker.ComputeUtilization = 0
		}

		return
	}
}

func benchmarkDurationSteps(
	workload cluster.Workload,
) int {
	if workload.RequiredMemoryMB >= 16000 {
		return 10
	}

	return 8
}
