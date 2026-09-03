package cluster

import "time"

type WorkerHealthConfig struct {
	SuspectedAfter time.Duration
	UnhealthyAfter time.Duration
}

func DefaultWorkerHealthConfig() WorkerHealthConfig {
	return WorkerHealthConfig{
		SuspectedAfter: 15 * time.Second,
		UnhealthyAfter: 30 * time.Second,
	}
}

func EvaluateWorkerHealth(
	worker Worker,
	now time.Time,
	config WorkerHealthConfig,
) WorkerState {
	if worker.State == WorkerDraining {
		return WorkerDraining
	}

	if worker.LastHeartbeat.IsZero() {
		return WorkerUnhealthy
	}

	sinceHeartbeat := now.Sub(worker.LastHeartbeat)

	if sinceHeartbeat >= config.UnhealthyAfter {
		return WorkerUnhealthy
	}

	if sinceHeartbeat >= config.SuspectedAfter {
		return WorkerSuspected
	}

	return WorkerHealthy
}
