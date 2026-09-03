package cluster

import (
	"context"
	"errors"
)

var (
	ErrWorkerNotFound      = errors.New("worker not found")
	ErrWorkerAlreadyExists = errors.New("worker already exists")

	ErrWorkloadNotFound      = errors.New("workload not found")
	ErrWorkloadAlreadyExists = errors.New("workload already exists")

	ErrPlacementConflict = errors.New("placement conflict")
)

type StateStore interface {
	RegisterWorker(ctx context.Context, worker Worker) error
	GetWorker(ctx context.Context, workerID string) (Worker, error)
	ListWorkers(ctx context.Context) ([]Worker, error)
	UpdateWorker(ctx context.Context, worker Worker) error

	CreateWorkload(ctx context.Context, workload Workload) error
	GetWorkload(ctx context.Context, workloadID string) (Workload, error)
	ListWorkloads(ctx context.Context) ([]Workload, error)
	UpdateWorkload(ctx context.Context, workload Workload) error

	Snapshot(ctx context.Context) (Snapshot, error)

	CommitPlacement(
		ctx context.Context,
		workloadID string,
		workerID string,
	) (Workload, error)

	ReleasePlacement(
		ctx context.Context,
		workloadID string,
		nextState WorkloadState,
	) (Workload, error)
}
