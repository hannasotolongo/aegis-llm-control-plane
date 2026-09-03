package scheduler

import (
	"testing"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
)

func TestSelectWorkerPrefersCachedModel(t *testing.T) {
	workload := cluster.Workload{
		ID:               "workload-locality",
		ModelID:          "llama-3-70b",
		RequiredMemoryMB: 40000,
		State:            cluster.WorkloadPending,
	}

	workers := []cluster.Worker{
		{
			ID:                 "worker-cached",
			AvailableMemoryMB:  50000,
			ComputeUtilization: 40,
			State:              cluster.WorkerHealthy,
			CachedModels:       []string{"llama-3-70b"},
		},
		{
			ID:                 "worker-uncached",
			AvailableMemoryMB:  70000,
			ComputeUtilization: 10,
			State:              cluster.WorkerHealthy,
		},
	}

	selected, err := SelectWorker(workload, workers)
	if err != nil {
		t.Fatalf("SelectWorker returned error: %v", err)
	}

	if selected.ID != "worker-cached" {
		t.Fatalf("expected worker-cached, got %q", selected.ID)
	}
}
