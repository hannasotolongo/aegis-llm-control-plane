package simulator

import (
	"context"
	"fmt"
	"time"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
)

type Simulator struct {
	store    cluster.StateStore
	scenario Scenario
}

func New(store cluster.StateStore) *Simulator {
	return NewWithScenario(
		store,
		ScenarioRisingPressure,
	)
}

func NewWithScenario(
	store cluster.StateStore,
	scenario Scenario,
) *Simulator {
	return &Simulator{
		store:    store,
		scenario: scenario,
	}
}

func (s *Simulator) SeedWorkers(
	ctx context.Context,
	count int,
) error {
	for i := 0; i < count; i++ {
		nodeID := fmt.Sprintf(
			"node-%d",
			i+1,
		)

		rackID := fmt.Sprintf(
			"rack-%d",
			(i%2)+1,
		)

		worker := cluster.Worker{
			ID:                  fmt.Sprintf("worker-%d", i+1),
			NodeID:              nodeID,
			GPUType:             "NVIDIA-H100",
			TotalMemoryMB:       81920,
			AvailableMemoryMB:   81920,
			ComputeUtilization:  0,
			MemoryUtilization:   0,
			ActiveWorkloadCount: 0,
			State:               cluster.WorkerHealthy,
			LastHeartbeat:       time.Now(),

			Topology: cluster.GPUTopology{
				NodeID:   nodeID,
				RackID:   rackID,
				GPUIndex: i,
				NVLinkDomain: fmt.Sprintf(
					"nvlink-%d",
					(i/2)+1,
				),
				Interconnect: cluster.InterconnectNVLink,
			},

			TopologyDomain: rackID,
		}

		if err := s.store.RegisterWorker(
			ctx,
			worker,
		); err != nil {
			return err
		}
	}

	return nil
}

func (s *Simulator) SeedWorkloads(
	ctx context.Context,
) error {
	workloads := []cluster.Workload{
		{
			ID:               "workload-1",
			ModelID:          "llama-3-8b",
			ArrivalTime:      time.Now(),
			Priority:         cluster.PriorityStandard,
			PromptTokens:     2048,
			MaxOutputTokens:  512,
			BatchSize:        1,
			RequiredMemoryMB: 18000,
			KVCacheMemoryMB:  2000,
			EstimatedCompute: 0.55,
			ExpectedDuration: 2 * time.Minute,
			LatencySLO:       500 * time.Millisecond,
			Checkpointable:   true,
			State:            cluster.WorkloadQueued,
		},
		{
			ID:               "workload-2",
			ModelID:          "mistral-7b",
			ArrivalTime:      time.Now(),
			Priority:         cluster.PriorityCritical,
			PromptTokens:     8192,
			MaxOutputTokens:  1024,
			BatchSize:        4,
			RequiredMemoryMB: 24000,
			KVCacheMemoryMB:  6000,
			EstimatedCompute: 0.75,
			ExpectedDuration: 90 * time.Second,
			LatencySLO:       250 * time.Millisecond,
			Checkpointable:   true,
			State:            cluster.WorkloadQueued,
		},
		{
			ID:               "workload-3",
			ModelID:          "vision-model",
			ArrivalTime:      time.Now(),
			Priority:         cluster.PriorityBatch,
			PromptTokens:     1024,
			MaxOutputTokens:  256,
			BatchSize:        8,
			RequiredMemoryMB: 12000,
			KVCacheMemoryMB:  1000,
			EstimatedCompute: 0.40,
			ExpectedDuration: 3 * time.Minute,
			LatencySLO:       time.Second,
			Checkpointable:   false,
			State:            cluster.WorkloadQueued,
		},
	}

	for i, workload := range workloads {
		if err := s.store.CreateWorkload(
			ctx,
			workload,
		); err != nil {
			return err
		}

		workerID := fmt.Sprintf(
			"worker-%d",
			i+1,
		)

		placed, err := s.store.CommitPlacement(
			ctx,
			workload.ID,
			workerID,
		)
		if err != nil {
			return err
		}

		placed.State =
			cluster.WorkloadRunning

		if err := s.store.UpdateWorkload(
			ctx,
			placed,
		); err != nil {
			return err
		}
	}

	return nil
}

func (s *Simulator) Run(
	ctx context.Context,
	interval time.Duration,
) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	lifecycle := NewWorkloadLifecycle()

	if err := s.InitializeLifecycle(
		ctx,
		lifecycle,
		time.Now(),
	); err != nil {
		return err
	}

	step := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case now := <-ticker.C:
			if err := s.updateWorkers(
				ctx,
				step,
			); err != nil {
				return err
			}

			if err := s.AdvanceWorkloads(
				ctx,
				lifecycle,
				now,
			); err != nil {
				return err
			}

			step++
		}
	}
}

func (s *Simulator) updateWorkers(
	ctx context.Context,
	step int,
) error {
	workers, err := s.store.ListWorkers(ctx)
	if err != nil {
		return err
	}

	for i, worker := range workers {
		utilization, err := s.scenario.Utilization(
			step,
			i,
		)
		if err != nil {
			return err
		}

		worker.ComputeUtilization =
			utilization.Compute

		worker.MemoryUtilization =
			utilization.Memory

		worker.LastHeartbeat =
			time.Now()

		if err := s.store.UpdateWorker(
			ctx,
			worker,
		); err != nil {
			return err
		}
	}

	return nil
}
