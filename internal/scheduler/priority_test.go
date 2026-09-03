package scheduler

import (
	"testing"
	"time"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
)

func TestOrderWorkloadsByPriority(t *testing.T) {
	workloads := []cluster.Workload{
		{ID: "batch", Priority: cluster.PriorityBatch},
		{ID: "standard", Priority: cluster.PriorityStandard},
		{ID: "critical", Priority: cluster.PriorityCritical},
	}

	ordered := OrderWorkloads(workloads)

	expected := []string{"critical", "standard", "batch"}

	for i, id := range expected {
		if ordered[i].ID != id {
			t.Fatalf(
				"position %d: expected %q, got %q",
				i,
				id,
				ordered[i].ID,
			)
		}
	}
}

func TestPriorityWeight(t *testing.T) {
	if PriorityWeight(cluster.PriorityCritical) <=
		PriorityWeight(cluster.PriorityStandard) {
		t.Fatal("critical priority must outrank standard")
	}

	if PriorityWeight(cluster.PriorityStandard) <=
		PriorityWeight(cluster.PriorityBatch) {
		t.Fatal("standard priority must outrank batch")
	}
}

func TestOrderWorkloadsWithAgingPreventsStarvation(t *testing.T) {
	now := time.Now()

	workloads := []cluster.Workload{
		{
			ID:          "critical-new",
			Priority:    cluster.PriorityCritical,
			ArrivalTime: now.Add(-1 * time.Minute),
		},
		{
			ID:          "batch-old",
			Priority:    cluster.PriorityBatch,
			ArrivalTime: now.Add(-20 * time.Minute),
		},
	}

	ordered := OrderWorkloadsWithAging(workloads, now)

	if len(ordered) != 2 {
		t.Fatalf(
			"expected 2 workloads, got %d",
			len(ordered),
		)
	}

	if ordered[0].ID != "batch-old" {
		t.Fatalf(
			"expected aged batch workload first, got %q",
			ordered[0].ID,
		)
	}
}
