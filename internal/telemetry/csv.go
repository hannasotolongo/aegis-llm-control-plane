package telemetry

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
)

func WriteCSV(
	writer io.Writer,
	records []Record,
) error {
	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	header := []string{
		"timestamp",
		"worker_id",
		"worker_state",
		"available_memory_mb",
		"total_memory_mb",
		"compute_utilization",
		"memory_utilization",
		"active_workload_count",
		"topology_domain",
		"workload_id",
		"workload_state",
		"model_id",
		"priority",
		"required_memory_mb",
		"estimated_compute",
		"latency_slo_ms",
		"checkpointable",
		"assigned_worker_id",
		"memory_pressure",
		"compute_pressure",
		"contention",
	}

	if err := csvWriter.Write(header); err != nil {
		return fmt.Errorf("write CSV header: %w", err)
	}

	for _, record := range records {
		label := LabelPressure(record)

		row := []string{
			record.Timestamp.UTC().Format("2006-01-02T15:04:05.000000000Z"),
			record.WorkerID,
			string(record.WorkerState),
			strconv.FormatUint(record.AvailableMemoryMB, 10),
			strconv.FormatUint(record.TotalMemoryMB, 10),
			strconv.FormatFloat(record.ComputeUtilization, 'f', -1, 64),
			strconv.FormatFloat(record.MemoryUtilization, 'f', -1, 64),
			strconv.Itoa(record.ActiveWorkloadCount),
			record.TopologyDomain,
			record.WorkloadID,
			string(record.WorkloadState),
			record.ModelID,
			string(record.Priority),
			strconv.FormatUint(record.RequiredMemoryMB, 10),
			strconv.FormatFloat(record.EstimatedCompute, 'f', -1, 64),
			strconv.FormatInt(record.LatencySLO.Milliseconds(), 10),
			strconv.FormatBool(record.Checkpointable),
			record.AssignedWorkerID,
			strconv.FormatBool(label.MemoryPressure),
			strconv.FormatBool(label.ComputePressure),
			strconv.FormatBool(label.Contention),
		}

		if err := csvWriter.Write(row); err != nil {
			return fmt.Errorf("write CSV record: %w", err)
		}
	}

	csvWriter.Flush()

	if err := csvWriter.Error(); err != nil {
		return fmt.Errorf("flush CSV: %w", err)
	}

	return nil
}
