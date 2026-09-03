package scheduler

import (
	"sort"
	"time"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
)

const workloadAgingInterval = 5 * time.Minute

func PriorityWeight(priority cluster.Priority) int {
	switch priority {
	case cluster.PriorityCritical:
		return 3
	case cluster.PriorityStandard:
		return 2
	case cluster.PriorityBatch:
		return 1
	default:
		return 0
	}
}

func EffectivePriorityWeight(
	workload cluster.Workload,
	now time.Time,
) int {
	weight := PriorityWeight(workload.Priority)

	if workload.ArrivalTime.IsZero() ||
		!now.After(workload.ArrivalTime) {
		return weight
	}

	waitTime := now.Sub(workload.ArrivalTime)
	agingBoost := int(waitTime / workloadAgingInterval)

	return weight + agingBoost
}

func OrderWorkloads(workloads []cluster.Workload) []cluster.Workload {
	ordered := append([]cluster.Workload(nil), workloads...)

	sort.SliceStable(ordered, func(i, j int) bool {
		wi := PriorityWeight(ordered[i].Priority)
		wj := PriorityWeight(ordered[j].Priority)

		if wi != wj {
			return wi > wj
		}

		return ordered[i].ArrivalTime.Before(ordered[j].ArrivalTime)
	})

	return ordered
}

func OrderWorkloadsWithAging(
	workloads []cluster.Workload,
	now time.Time,
) []cluster.Workload {
	ordered := append([]cluster.Workload(nil), workloads...)

	sort.SliceStable(ordered, func(i, j int) bool {
		wi := EffectivePriorityWeight(ordered[i], now)
		wj := EffectivePriorityWeight(ordered[j], now)

		if wi != wj {
			return wi > wj
		}

		return ordered[i].ArrivalTime.Before(ordered[j].ArrivalTime)
	})

	return ordered
}
