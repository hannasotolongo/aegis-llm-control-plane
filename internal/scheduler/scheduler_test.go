package scheduler

import (
	"errors"
	"testing"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
)

func TestSelectWorker(t *testing.T) {
	workload := cluster.Workload{
		ID:                     "workload-1",
		ModelID:                "llama-3-70b",
		Priority:               cluster.PriorityStandard,
		RequiredMemoryMB:       40000,
		RequiredTopologyDomain: "zone-a",
		State:                  cluster.WorkloadPending,
	}

	workers := []cluster.Worker{
		{
			ID:                 "worker-1",
			AvailableMemoryMB:  60000,
			ComputeUtilization: 70,
			State:              cluster.WorkerHealthy,
			TopologyDomain:     "zone-a",
		},
		{
			ID:                 "worker-2",
			AvailableMemoryMB:  70000,
			ComputeUtilization: 25,
			State:              cluster.WorkerHealthy,
			TopologyDomain:     "zone-a",
		},
	}

	selected, err := SelectWorker(workload, workers)
	if err != nil {
		t.Fatalf("SelectWorker returned error: %v", err)
	}

	if selected.ID != "worker-2" {
		t.Fatalf("expected worker-2, got %q", selected.ID)
	}
}

func TestSelectWorkerUsesStructuredRackTopology(t *testing.T) {
	workload := cluster.Workload{
		ID:                     "workload-1",
		ModelID:                "llama-3-70b",
		RequiredMemoryMB:       20000,
		RequiredTopologyDomain: "rack-1",
		State:                  cluster.WorkloadPending,
	}

	workers := []cluster.Worker{
		{
			ID:                 "worker-wrong-rack",
			AvailableMemoryMB:  60000,
			ComputeUtilization: 5,
			State:              cluster.WorkerHealthy,
			Topology: cluster.GPUTopology{
				RackID: "rack-2",
			},
		},
		{
			ID:                 "worker-right-rack",
			AvailableMemoryMB:  60000,
			ComputeUtilization: 50,
			State:              cluster.WorkerHealthy,
			Topology: cluster.GPUTopology{
				RackID: "rack-1",
			},
		},
	}

	selected, err := SelectWorker(workload, workers)
	if err != nil {
		t.Fatalf("SelectWorker returned error: %v", err)
	}

	if selected.ID != "worker-right-rack" {
		t.Fatalf(
			"expected worker-right-rack, got %q",
			selected.ID,
		)
	}
}

func TestSelectWorkerRejectsStructuredTopologyMismatch(t *testing.T) {
	workload := cluster.Workload{
		ID:                     "workload-1",
		RequiredMemoryMB:       1000,
		RequiredTopologyDomain: "rack-1",
		State:                  cluster.WorkloadPending,
	}

	workers := []cluster.Worker{
		{
			ID:                "worker-1",
			AvailableMemoryMB: 81920,
			State:             cluster.WorkerHealthy,
			Topology: cluster.GPUTopology{
				RackID: "rack-2",
			},
		},
	}

	_, err := SelectWorker(workload, workers)
	if !errors.Is(err, ErrNoEligibleWorker) {
		t.Fatalf(
			"expected ErrNoEligibleWorker for topology mismatch, got %v",
			err,
		)
	}
}

func TestSelectWorkerFallsBackToLegacyTopologyDomain(t *testing.T) {
	workload := cluster.Workload{
		ID:                     "workload-1",
		RequiredMemoryMB:       1000,
		RequiredTopologyDomain: "zone-a",
		State:                  cluster.WorkloadPending,
	}

	workers := []cluster.Worker{
		{
			ID:                "worker-1",
			AvailableMemoryMB: 81920,
			State:             cluster.WorkerHealthy,
			TopologyDomain:    "zone-a",
		},
	}

	selected, err := SelectWorker(workload, workers)
	if err != nil {
		t.Fatalf("SelectWorker returned error: %v", err)
	}

	if selected.ID != "worker-1" {
		t.Fatalf("expected worker-1, got %q", selected.ID)
	}
}

func TestSelectWorkerMatchesRequiredNode(t *testing.T) {
	workload := cluster.Workload{
		ID:               "workload-node",
		RequiredMemoryMB: 1000,
		State:            cluster.WorkloadPending,
		TopologyRequirement: cluster.TopologyRequirement{
			RequiredNodeID: "node-2",
		},
	}

	workers := []cluster.Worker{
		{
			ID:                "worker-1",
			AvailableMemoryMB: 81920,
			State:             cluster.WorkerHealthy,
			Topology: cluster.GPUTopology{
				NodeID: "node-1",
			},
		},
		{
			ID:                "worker-2",
			AvailableMemoryMB: 81920,
			State:             cluster.WorkerHealthy,
			Topology: cluster.GPUTopology{
				NodeID: "node-2",
			},
		},
	}

	selected, err := SelectWorker(workload, workers)
	if err != nil {
		t.Fatalf("SelectWorker returned error: %v", err)
	}

	if selected.ID != "worker-2" {
		t.Fatalf("expected worker-2, got %q", selected.ID)
	}
}

func TestSelectWorkerMatchesRequiredRack(t *testing.T) {
	workload := cluster.Workload{
		ID:               "workload-rack",
		RequiredMemoryMB: 1000,
		State:            cluster.WorkloadPending,
		TopologyRequirement: cluster.TopologyRequirement{
			RequiredRackID: "rack-2",
		},
	}

	workers := []cluster.Worker{
		{
			ID:                "worker-1",
			AvailableMemoryMB: 81920,
			State:             cluster.WorkerHealthy,
			Topology: cluster.GPUTopology{
				RackID: "rack-1",
			},
		},
		{
			ID:                "worker-2",
			AvailableMemoryMB: 81920,
			State:             cluster.WorkerHealthy,
			Topology: cluster.GPUTopology{
				RackID: "rack-2",
			},
		},
	}

	selected, err := SelectWorker(workload, workers)
	if err != nil {
		t.Fatalf("SelectWorker returned error: %v", err)
	}

	if selected.ID != "worker-2" {
		t.Fatalf("expected worker-2, got %q", selected.ID)
	}
}

func TestSelectWorkerMatchesRequiredNVLinkDomain(t *testing.T) {
	workload := cluster.Workload{
		ID:               "workload-nvlink",
		RequiredMemoryMB: 1000,
		State:            cluster.WorkloadPending,
		TopologyRequirement: cluster.TopologyRequirement{
			RequiredNVLinkDomain: "nvlink-2",
		},
	}

	workers := []cluster.Worker{
		{
			ID:                "worker-1",
			AvailableMemoryMB: 81920,
			State:             cluster.WorkerHealthy,
			Topology: cluster.GPUTopology{
				NVLinkDomain: "nvlink-1",
			},
		},
		{
			ID:                "worker-2",
			AvailableMemoryMB: 81920,
			State:             cluster.WorkerHealthy,
			Topology: cluster.GPUTopology{
				NVLinkDomain: "nvlink-2",
			},
		},
	}

	selected, err := SelectWorker(workload, workers)
	if err != nil {
		t.Fatalf("SelectWorker returned error: %v", err)
	}

	if selected.ID != "worker-2" {
		t.Fatalf("expected worker-2, got %q", selected.ID)
	}
}

func TestSelectWorkerMatchesRequiredInterconnect(t *testing.T) {
	workload := cluster.Workload{
		ID:               "workload-interconnect",
		RequiredMemoryMB: 1000,
		State:            cluster.WorkloadPending,
		TopologyRequirement: cluster.TopologyRequirement{
			RequiredInterconnect: cluster.InterconnectNVLink,
		},
	}

	workers := []cluster.Worker{
		{
			ID:                "worker-pcie",
			AvailableMemoryMB: 81920,
			State:             cluster.WorkerHealthy,
			Topology: cluster.GPUTopology{
				Interconnect: cluster.InterconnectPCIe,
			},
		},
		{
			ID:                "worker-nvlink",
			AvailableMemoryMB: 81920,
			State:             cluster.WorkerHealthy,
			Topology: cluster.GPUTopology{
				Interconnect: cluster.InterconnectNVLink,
			},
		},
	}

	selected, err := SelectWorker(workload, workers)
	if err != nil {
		t.Fatalf("SelectWorker returned error: %v", err)
	}

	if selected.ID != "worker-nvlink" {
		t.Fatalf(
			"expected worker-nvlink, got %q",
			selected.ID,
		)
	}
}

func TestSelectWorkerRejectsInsufficientMemory(t *testing.T) {
	workload := cluster.Workload{
		ID:               "workload-1",
		RequiredMemoryMB: 90000,
	}

	workers := []cluster.Worker{
		{
			ID:                "worker-1",
			AvailableMemoryMB: 81920,
			State:             cluster.WorkerHealthy,
		},
	}

	_, err := SelectWorker(workload, workers)
	if !errors.Is(err, ErrNoEligibleWorker) {
		t.Fatalf("expected ErrNoEligibleWorker, got %v", err)
	}
}

func TestSelectWorkerRejectsUnhealthyWorker(t *testing.T) {
	workload := cluster.Workload{
		ID:               "workload-1",
		RequiredMemoryMB: 1000,
	}

	workers := []cluster.Worker{
		{
			ID:                "worker-1",
			AvailableMemoryMB: 81920,
			State:             cluster.WorkerUnhealthy,
		},
	}

	_, err := SelectWorker(workload, workers)
	if !errors.Is(err, ErrNoEligibleWorker) {
		t.Fatalf("expected ErrNoEligibleWorker, got %v", err)
	}
}

func TestSelectWorkerPrefersLowerUtilization(t *testing.T) {
	workload := cluster.Workload{
		ID:               "workload-1",
		ModelID:          "llama-3-70b",
		RequiredMemoryMB: 20000,
		State:            cluster.WorkloadPending,
	}

	workers := []cluster.Worker{
		{
			ID:                 "worker-busy",
			AvailableMemoryMB:  60000,
			ComputeUtilization: 80,
			MemoryUtilization:  70,
			State:              cluster.WorkerHealthy,
		},
		{
			ID:                 "worker-idle",
			AvailableMemoryMB:  60000,
			ComputeUtilization: 20,
			MemoryUtilization:  20,
			State:              cluster.WorkerHealthy,
		},
	}

	selected, err := SelectWorker(workload, workers)
	if err != nil {
		t.Fatalf("SelectWorker returned error: %v", err)
	}

	if selected.ID != "worker-idle" {
		t.Fatalf(
			"expected worker-idle due to lower utilization, got %q",
			selected.ID,
		)
	}
}

func TestSelectWorkerPrefersFewerActiveWorkloads(t *testing.T) {
	workload := cluster.Workload{
		ID:               "workload-1",
		ModelID:          "llama-3-70b",
		RequiredMemoryMB: 20000,
		State:            cluster.WorkloadPending,
	}

	workers := []cluster.Worker{
		{
			ID:                  "worker-loaded",
			AvailableMemoryMB:   60000,
			ComputeUtilization:  30,
			MemoryUtilization:   30,
			ActiveWorkloadCount: 5,
			State:               cluster.WorkerHealthy,
		},
		{
			ID:                  "worker-light",
			AvailableMemoryMB:   60000,
			ComputeUtilization:  30,
			MemoryUtilization:   30,
			ActiveWorkloadCount: 1,
			State:               cluster.WorkerHealthy,
		},
	}

	selected, err := SelectWorker(workload, workers)
	if err != nil {
		t.Fatalf("SelectWorker returned error: %v", err)
	}

	if selected.ID != "worker-light" {
		t.Fatalf(
			"expected worker-light due to fewer active workloads, got %q",
			selected.ID,
		)
	}
}

func TestExplainWorkerScore(t *testing.T) {
	workload := cluster.Workload{
		ID:      "workload-1",
		ModelID: "llama-3-70b",
	}

	worker := cluster.Worker{
		ID:                  "worker-1",
		AvailableMemoryMB:   51200,
		ComputeUtilization:  20,
		MemoryUtilization:   10,
		ActiveWorkloadCount: 2,
		CachedModels:        []string{"llama-3-70b"},
	}

	breakdown := ExplainWorkerScore(workload, worker)

	if breakdown.MemoryHeadroom != 50 {
		t.Fatalf(
			"expected memory headroom 50, got %v",
			breakdown.MemoryHeadroom,
		)
	}

	if breakdown.ComputePenalty != 20 {
		t.Fatalf(
			"expected compute penalty 20, got %v",
			breakdown.ComputePenalty,
		)
	}

	if breakdown.MemoryPenalty != 10 {
		t.Fatalf(
			"expected memory penalty 10, got %v",
			breakdown.MemoryPenalty,
		)
	}

	if breakdown.ActiveWorkloadPenalty != 20 {
		t.Fatalf(
			"expected active workload penalty 20, got %v",
			breakdown.ActiveWorkloadPenalty,
		)
	}

	if breakdown.ModelLocalityBonus != 100 {
		t.Fatalf(
			"expected model locality bonus 100, got %v",
			breakdown.ModelLocalityBonus,
		)
	}

	if breakdown.Total != 100 {
		t.Fatalf(
			"expected total score 100, got %v",
			breakdown.Total,
		)
	}
}

func TestSelectWorkerBreaksExactScoreTieByWorkerID(t *testing.T) {
	workload := cluster.Workload{
		ID:               "workload-1",
		ModelID:          "llama-3",
		RequiredMemoryMB: 1000,
		State:            cluster.WorkloadPending,
	}

	workers := []cluster.Worker{
		{
			ID:                  "worker-b",
			AvailableMemoryMB:   64000,
			ComputeUtilization:  20,
			MemoryUtilization:   20,
			ActiveWorkloadCount: 1,
			State:               cluster.WorkerHealthy,
		},
		{
			ID:                  "worker-a",
			AvailableMemoryMB:   64000,
			ComputeUtilization:  20,
			MemoryUtilization:   20,
			ActiveWorkloadCount: 1,
			State:               cluster.WorkerHealthy,
		},
	}

	selected, err := SelectWorker(workload, workers)
	if err != nil {
		t.Fatalf("SelectWorker returned error: %v", err)
	}

	if selected.ID != "worker-a" {
		t.Fatalf(
			"expected deterministic tie-break to select worker-a, got %q",
			selected.ID,
		)
	}
}
