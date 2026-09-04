package cluster

import (
	"context"
	"fmt"
	"sync"
)

type InMemoryStateStore struct {
	mu sync.RWMutex

	workers   map[string]Worker
	workloads map[string]Workload
}

func NewInMemoryStateStore() *InMemoryStateStore {
	return &InMemoryStateStore{
		workers:   make(map[string]Worker),
		workloads: make(map[string]Workload),
	}
}

func cloneWorker(worker Worker) Worker {
	cloned := worker

	if worker.CachedModels != nil {
		cloned.CachedModels = append([]string(nil), worker.CachedModels...)
	}

	return cloned
}

func cloneWorkload(workload Workload) Workload {
	return workload
}

func (s *InMemoryStateStore) RegisterWorker(
	ctx context.Context,
	worker Worker,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := ValidateWorker(worker); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.workers[worker.ID]; exists {
		return fmt.Errorf(
			"%w: %s",
			ErrWorkerAlreadyExists,
			worker.ID,
		)
	}

	s.workers[worker.ID] = cloneWorker(worker)

	return nil
}

func (s *InMemoryStateStore) GetWorker(
	ctx context.Context,
	workerID string,
) (Worker, error) {
	if err := ctx.Err(); err != nil {
		return Worker{}, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	worker, exists := s.workers[workerID]
	if !exists {
		return Worker{}, fmt.Errorf(
			"%w: %s",
			ErrWorkerNotFound,
			workerID,
		)
	}

	return cloneWorker(worker), nil
}

func (s *InMemoryStateStore) ListWorkers(
	ctx context.Context,
) ([]Worker, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	workers := make([]Worker, 0, len(s.workers))

	for _, worker := range s.workers {
		workers = append(workers, cloneWorker(worker))
	}

	return workers, nil
}

func (s *InMemoryStateStore) UpdateWorker(
	ctx context.Context,
	worker Worker,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := ValidateWorker(worker); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.workers[worker.ID]; !exists {
		return fmt.Errorf(
			"%w: %s",
			ErrWorkerNotFound,
			worker.ID,
		)
	}

	s.workers[worker.ID] = cloneWorker(worker)

	return nil
}

func (s *InMemoryStateStore) CreateWorkload(
	ctx context.Context,
	workload Workload,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := ValidateWorkload(workload); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.workloads[workload.ID]; exists {
		return fmt.Errorf(
			"%w: %s",
			ErrWorkloadAlreadyExists,
			workload.ID,
		)
	}

	s.workloads[workload.ID] = cloneWorkload(workload)

	return nil
}
func (s *InMemoryStateStore) GetWorkload(
	ctx context.Context,
	workloadID string,
) (Workload, error) {
	if err := ctx.Err(); err != nil {
		return Workload{}, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	workload, exists := s.workloads[workloadID]
	if !exists {
		return Workload{}, fmt.Errorf(
			"%w: %s",
			ErrWorkloadNotFound,
			workloadID,
		)
	}

	return cloneWorkload(workload), nil
}

func (s *InMemoryStateStore) ListWorkloads(
	ctx context.Context,
) ([]Workload, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	workloads := make([]Workload, 0, len(s.workloads))

	for _, workload := range s.workloads {
		workloads = append(workloads, cloneWorkload(workload))
	}

	return workloads, nil
}

func (s *InMemoryStateStore) UpdateWorkload(
	ctx context.Context,
	workload Workload,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := ValidateWorkload(workload); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.workloads[workload.ID]; !exists {
		return fmt.Errorf(
			"%w: %s",
			ErrWorkloadNotFound,
			workload.ID,
		)
	}

	s.workloads[workload.ID] = cloneWorkload(workload)

	return nil
}
func (s *InMemoryStateStore) Snapshot(
	ctx context.Context,
) (Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	workers := make([]Worker, 0, len(s.workers))

	for _, worker := range s.workers {
		workers = append(workers, cloneWorker(worker))
	}

	workloads := make([]Workload, 0, len(s.workloads))

	for _, workload := range s.workloads {
		workloads = append(workloads, cloneWorkload(workload))
	}

	return Snapshot{
		Workers:   workers,
		Workloads: workloads,
	}, nil
}

func (s *InMemoryStateStore) CommitPlacement(
	ctx context.Context,
	workloadID string,
	workerID string,
) (Workload, error) {
	if err := ctx.Err(); err != nil {
		return Workload{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	workload, exists := s.workloads[workloadID]
	if !exists {
		return Workload{}, fmt.Errorf(
			"%w: %s",
			ErrWorkloadNotFound,
			workloadID,
		)
	}

	worker, exists := s.workers[workerID]
	if !exists {
		return Workload{}, fmt.Errorf(
			"%w: %s",
			ErrWorkerNotFound,
			workerID,
		)
	}

	if worker.State != WorkerHealthy {
		return Workload{}, fmt.Errorf(
			"%w: worker %q is not healthy",
			ErrPlacementConflict,
			worker.ID,
		)
	}

	if !MatchesTopologyRequirement(workload, worker) {
		return Workload{}, fmt.Errorf(
			"%w: worker %q no longer satisfies topology requirements",
			ErrPlacementConflict,
			worker.ID,
		)
	}

	if workload.State != WorkloadPending &&
		workload.State != WorkloadQueued {
		return Workload{}, fmt.Errorf(
			"%w: workload %q is not schedulable from state %q",
			ErrPlacementConflict,
			workload.ID,
			workload.State,
		)
	}

	if worker.AvailableMemoryMB < workload.RequiredMemoryMB {
		return Workload{}, fmt.Errorf(
			"%w: worker %q has %d MB available, workload requires %d MB",
			ErrPlacementConflict,
			worker.ID,
			worker.AvailableMemoryMB,
			workload.RequiredMemoryMB,
		)
	}

	worker.AvailableMemoryMB -= workload.RequiredMemoryMB
	worker.ActiveWorkloadCount++

	workload.AssignedWorkerID = worker.ID
	workload.State = WorkloadPlaced

	s.workers[worker.ID] = cloneWorker(worker)
	s.workloads[workload.ID] = cloneWorkload(workload)

	return cloneWorkload(workload), nil
}
func (s *InMemoryStateStore) ReleasePlacement(
	ctx context.Context,
	workloadID string,
	nextState WorkloadState,
) (Workload, error) {
	if err := ctx.Err(); err != nil {
		return Workload{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	workload, exists := s.workloads[workloadID]
	if !exists {
		return Workload{}, fmt.Errorf(
			"%w: %s",
			ErrWorkloadNotFound,
			workloadID,
		)
	}

	if workload.AssignedWorkerID == "" {
		return Workload{}, fmt.Errorf(
			"%w: workload %q has no assigned worker",
			ErrPlacementConflict,
			workload.ID,
		)
	}

	worker, exists := s.workers[workload.AssignedWorkerID]
	if !exists {
		return Workload{}, fmt.Errorf(
			"%w: %s",
			ErrWorkerNotFound,
			workload.AssignedWorkerID,
		)
	}
	switch workload.State {
	case WorkloadPlaced, WorkloadRunning:
	default:
		return Workload{}, fmt.Errorf(
			"%w: workload %q cannot release placement from state %q",
			ErrPlacementConflict,
			workload.ID,
			workload.State,
		)
	}

	switch nextState {
	case WorkloadQueued,
		WorkloadRecovering,
		WorkloadFailed:
	default:
		return Workload{}, fmt.Errorf(
			"%w: invalid release target state %q",
			ErrPlacementConflict,
			nextState,
		)
	}

	releasedMemory := workload.RequiredMemoryMB

	if releasedMemory > worker.TotalMemoryMB-worker.AvailableMemoryMB {
		worker.AvailableMemoryMB = worker.TotalMemoryMB
	} else {
		worker.AvailableMemoryMB += releasedMemory
	}

	if worker.ActiveWorkloadCount > 0 {
		worker.ActiveWorkloadCount--
	}

	workload.AssignedWorkerID = ""
	workload.State = nextState

	if err := ValidateWorker(worker); err != nil {
		return Workload{}, err
	}

	if err := ValidateWorkload(workload); err != nil {
		return Workload{}, err
	}

	s.workers[worker.ID] = cloneWorker(worker)
	s.workloads[workload.ID] = cloneWorkload(workload)

	return cloneWorkload(workload), nil
}

var _ StateStore = (*InMemoryStateStore)(nil)
