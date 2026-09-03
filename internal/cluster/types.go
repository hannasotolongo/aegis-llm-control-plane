package cluster

import "time"

type WorkerState string

const (
	WorkerHealthy   WorkerState = "HEALTHY"
	WorkerSuspected WorkerState = "SUSPECTED"
	WorkerUnhealthy WorkerState = "UNHEALTHY"
	WorkerDraining  WorkerState = "DRAINING"
)

type WorkloadState string

const (
	WorkloadPending    WorkloadState = "PENDING"
	WorkloadQueued     WorkloadState = "QUEUED"
	WorkloadPlaced     WorkloadState = "PLACED"
	WorkloadRunning    WorkloadState = "RUNNING"
	WorkloadPreempted  WorkloadState = "PREEMPTED"
	WorkloadRecovering WorkloadState = "RECOVERING"
	WorkloadCompleted  WorkloadState = "COMPLETED"
	WorkloadFailed     WorkloadState = "FAILED"
)

type Priority string

const (
	PriorityCritical Priority = "CRITICAL"
	PriorityStandard Priority = "STANDARD"
	PriorityBatch    Priority = "BATCH"
)

type Worker struct {
	ID                  string
	NodeID              string
	GPUType             string
	TotalMemoryMB       uint64
	AvailableMemoryMB   uint64
	ComputeUtilization  float64
	MemoryUtilization   float64
	ActiveWorkloadCount int
	State               WorkerState
	LastHeartbeat       time.Time
	TopologyDomain      string
	CachedModels        []string
}

type Workload struct {
	ID                     string
	ModelID                string
	ArrivalTime            time.Time
	Priority               Priority
	RequiredMemoryMB       uint64
	EstimatedCompute       float64
	ExpectedDuration       time.Duration
	LatencySLO             time.Duration
	Checkpointable         bool
	RequiredTopologyDomain string
	KVCacheKey             string
	State                  WorkloadState
	AssignedWorkerID       string
}
