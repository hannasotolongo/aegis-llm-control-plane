package cluster

import (
	"testing"
	"time"
)

func TestEvaluateWorkerHealth(t *testing.T) {
	now := time.Now()

	config := WorkerHealthConfig{
		SuspectedAfter: 10 * time.Second,
		UnhealthyAfter: 30 * time.Second,
	}

	tests := []struct {
		name     string
		worker   Worker
		expected WorkerState
	}{
		{
			name: "healthy recent heartbeat",
			worker: Worker{
				State:         WorkerHealthy,
				LastHeartbeat: now.Add(-5 * time.Second),
			},
			expected: WorkerHealthy,
		},
		{
			name: "suspected stale heartbeat",
			worker: Worker{
				State:         WorkerHealthy,
				LastHeartbeat: now.Add(-20 * time.Second),
			},
			expected: WorkerSuspected,
		},
		{
			name: "unhealthy stale heartbeat",
			worker: Worker{
				State:         WorkerHealthy,
				LastHeartbeat: now.Add(-45 * time.Second),
			},
			expected: WorkerUnhealthy,
		},
		{
			name: "missing heartbeat",
			worker: Worker{
				State: WorkerHealthy,
			},
			expected: WorkerUnhealthy,
		},
		{
			name: "draining worker stays draining",
			worker: Worker{
				State:         WorkerDraining,
				LastHeartbeat: now.Add(-1 * time.Hour),
			},
			expected: WorkerDraining,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := EvaluateWorkerHealth(
				tt.worker,
				now,
				config,
			)

			if actual != tt.expected {
				t.Fatalf(
					"expected %q, got %q",
					tt.expected,
					actual,
				)
			}
		})
	}
}
