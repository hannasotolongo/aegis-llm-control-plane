package scheduler

import "github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"

func matchesTopology(
	workload cluster.Workload,
	worker cluster.Worker,
) bool {
	return cluster.MatchesTopologyRequirement(
		workload,
		worker,
	)
}
