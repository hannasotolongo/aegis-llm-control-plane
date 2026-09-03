package telemetry

import (
	"time"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
)

type Record struct {
	Timestamp time.Time

	WorkerID            string
	WorkerState         cluster.WorkerState
	AvailableMemoryMB   uint64
	TotalMemoryMB       uint64
	ComputeUtilization  float64
	MemoryUtilization   float64
	ActiveWorkloadCount int
	TopologyDomain      string

	WorkloadID       string
	WorkloadState    cluster.WorkloadState
	ModelID          string
	Priority         cluster.Priority
	RequiredMemoryMB uint64
	EstimatedCompute float64
	LatencySLO       time.Duration
	Checkpointable   bool
	AssignedWorkerID string
}
