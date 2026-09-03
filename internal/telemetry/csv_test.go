package telemetry

import (
	"bytes"
	"encoding/csv"
	"testing"
	"time"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
)

func TestWriteCSV(t *testing.T) {
	timestamp := time.Date(
		2026,
		time.September,
		2,
		12,
		0,
		0,
		0,
		time.UTC,
	)

	records := []Record{
		{
			Timestamp:           timestamp,
			WorkerID:            "worker-1",
			WorkerState:         cluster.WorkerHealthy,
			AvailableMemoryMB:   60000,
			TotalMemoryMB:       80000,
			ComputeUtilization:  72.5,
			MemoryUtilization:   65.25,
			ActiveWorkloadCount: 2,
			TopologyDomain:      "rack-a",
			WorkloadID:          "workload-1",
			WorkloadState:       cluster.WorkloadRunning,
			ModelID:             "llama-3",
			Priority:            cluster.PriorityStandard,
			RequiredMemoryMB:    20000,
			EstimatedCompute:    0.8,
			LatencySLO:          1500 * time.Millisecond,
			Checkpointable:      true,
			AssignedWorkerID:    "worker-1",
		},
	}

	var buffer bytes.Buffer

	if err := WriteCSV(&buffer, records); err != nil {
		t.Fatalf("WriteCSV returned error: %v", err)
	}

	reader := csv.NewReader(&buffer)

	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	header := rows[0]
	data := rows[1]

	if header[0] != "timestamp" {
		t.Fatalf("expected timestamp header, got %q", header[0])
	}

	if data[1] != "worker-1" {
		t.Fatalf("expected worker-1, got %q", data[1])
	}

	if data[9] != "workload-1" {
		t.Fatalf("expected workload-1, got %q", data[9])
	}

	if data[11] != "llama-3" {
		t.Fatalf("expected llama-3, got %q", data[11])
	}

	if data[15] != "1500" {
		t.Fatalf("expected latency SLO 1500 ms, got %q", data[15])
	}

	if data[16] != "true" {
		t.Fatalf("expected checkpointable true, got %q", data[16])
	}
}
