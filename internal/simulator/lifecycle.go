package simulator

import (
	"context"
	"sync"
	"time"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
)

type ActiveWorkload struct {
	WorkloadID string
	WorkerID   string
	MemoryMB   uint64
	StartedAt  time.Time
	Duration   time.Duration
}

type WorkloadLifecycle struct {
	mu     sync.RWMutex
	active map[string]ActiveWorkload
}

func NewWorkloadLifecycle() *WorkloadLifecycle {
	return &WorkloadLifecycle{
		active: make(map[string]ActiveWorkload),
	}
}

func (l *WorkloadLifecycle) Track(
	workload cluster.Workload,
	start time.Time,
) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.active[workload.ID] = ActiveWorkload{
		WorkloadID: workload.ID,
		WorkerID:   workload.AssignedWorkerID,
		MemoryMB:   workload.RequiredMemoryMB,
		StartedAt:  start,
		Duration:   workload.ExpectedDuration,
	}
}

func (l *WorkloadLifecycle) Completed(
	workloadID string,
	now time.Time,
) bool {
	active, ok := l.activeWorkload(workloadID)
	if !ok {
		return false
	}

	return !now.Before(
		active.StartedAt.Add(active.Duration),
	)
}

func (l *WorkloadLifecycle) Remove(
	workloadID string,
) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.active, workloadID)
}

func (l *WorkloadLifecycle) activeWorkload(
	workloadID string,
) (ActiveWorkload, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	workload, ok := l.active[workloadID]

	return workload, ok
}

func (s *Simulator) InitializeLifecycle(
	ctx context.Context,
	lifecycle *WorkloadLifecycle,
	now time.Time,
) error {
	workloads, err := s.store.ListWorkloads(ctx)
	if err != nil {
		return err
	}

	for _, workload := range workloads {
		if workload.State != cluster.WorkloadRunning {
			continue
		}

		if workload.AssignedWorkerID == "" {
			continue
		}

		start := workload.ArrivalTime

		if start.IsZero() {
			start = now
		}

		lifecycle.Track(
			workload,
			start,
		)
	}

	return nil
}

func (s *Simulator) AdvanceWorkloads(
	ctx context.Context,
	lifecycle *WorkloadLifecycle,
	now time.Time,
) error {
	lifecycle.mu.RLock()

	activeWorkloads := make(
		[]ActiveWorkload,
		0,
		len(lifecycle.active),
	)

	for _, workload := range lifecycle.active {
		activeWorkloads = append(
			activeWorkloads,
			workload,
		)
	}

	lifecycle.mu.RUnlock()

	for _, active := range activeWorkloads {
		if !lifecycle.Completed(
			active.WorkloadID,
			now,
		) {
			continue
		}

		workload, err := s.store.GetWorkload(
			ctx,
			active.WorkloadID,
		)
		if err != nil {
			return err
		}

		if workload.State == cluster.WorkloadCompleted {
			lifecycle.Remove(active.WorkloadID)
			continue
		}

		if err := s.completeWorkload(
			ctx,
			workload,
		); err != nil {
			return err
		}

		lifecycle.Remove(active.WorkloadID)
	}

	return nil
}

func (s *Simulator) completeWorkload(
	ctx context.Context,
	workload cluster.Workload,
) error {
	worker, err := s.store.GetWorker(
		ctx,
		workload.AssignedWorkerID,
	)
	if err != nil {
		return err
	}

	if worker.AvailableMemoryMB+
		workload.RequiredMemoryMB >
		worker.TotalMemoryMB {
		worker.AvailableMemoryMB =
			worker.TotalMemoryMB
	} else {
		worker.AvailableMemoryMB +=
			workload.RequiredMemoryMB
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

	if worker.ComputeUtilization >= 10 {
		worker.ComputeUtilization -= 10
	} else {
		worker.ComputeUtilization = 0
	}

	workload.State = cluster.WorkloadCompleted

	if err := s.store.UpdateWorker(
		ctx,
		worker,
	); err != nil {
		return err
	}

	if err := s.store.UpdateWorkload(
		ctx,
		workload,
	); err != nil {
		return err
	}

	return nil
}
