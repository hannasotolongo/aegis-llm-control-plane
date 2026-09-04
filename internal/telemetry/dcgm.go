package telemetry

import (
	"errors"
	"fmt"
	"time"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
)

var (
	ErrInvalidDCGMSample = errors.New("invalid DCGM sample")
	ErrStaleDCGMSample   = errors.New("stale DCGM sample")
)

const DefaultDCGMMaxAge = 15 * time.Second

// DCGMSample represents the subset of NVIDIA DCGM telemetry
// that Aegis currently needs for placement decisions.
//
// The fields correspond to:
//
//	DCGM_FI_DEV_GPU_UTIL
//	DCGM_FI_DEV_FB_TOTAL
//	DCGM_FI_DEV_FB_FREE
//	DCGM_FI_DEV_FB_USED
//
// Values are normalized at this boundary before entering
// Aegis cluster state.
type DCGMSample struct {
	WorkerID string
	NodeID   string
	GPUType  string

	Timestamp time.Time

	GPUUtilizationPercent float64

	FramebufferTotalMB uint64
	FramebufferFreeMB  uint64
	FramebufferUsedMB  uint64
}

// ApplyDCGMSample validates a DCGM observation and applies its live
// resource information to an existing Aegis worker.
//
// Scheduling-owned metadata such as topology, cached models,
// active workload count, and worker identity remain under Aegis
// control and are not overwritten by telemetry.
func ApplyDCGMSample(
	worker cluster.Worker,
	sample DCGMSample,
	now time.Time,
	maxAge time.Duration,
) (cluster.Worker, error) {
	if worker.ID == "" {
		return cluster.Worker{}, fmt.Errorf(
			"%w: worker ID is empty",
			ErrInvalidDCGMSample,
		)
	}

	if sample.WorkerID == "" {
		return cluster.Worker{}, fmt.Errorf(
			"%w: sample worker ID is empty",
			ErrInvalidDCGMSample,
		)
	}

	if sample.WorkerID != worker.ID {
		return cluster.Worker{}, fmt.Errorf(
			"%w: sample worker %q does not match target worker %q",
			ErrInvalidDCGMSample,
			sample.WorkerID,
			worker.ID,
		)
	}

	if sample.Timestamp.IsZero() {
		return cluster.Worker{}, fmt.Errorf(
			"%w: timestamp is missing",
			ErrInvalidDCGMSample,
		)
	}

	if maxAge <= 0 {
		maxAge = DefaultDCGMMaxAge
	}

	if sample.Timestamp.After(now) {
		return cluster.Worker{}, fmt.Errorf(
			"%w: timestamp %s is in the future",
			ErrInvalidDCGMSample,
			sample.Timestamp.Format(time.RFC3339Nano),
		)
	}

	if now.Sub(sample.Timestamp) > maxAge {
		return cluster.Worker{}, fmt.Errorf(
			"%w: sample age %s exceeds maximum %s",
			ErrStaleDCGMSample,
			now.Sub(sample.Timestamp),
			maxAge,
		)
	}

	if sample.GPUUtilizationPercent < 0 ||
		sample.GPUUtilizationPercent > 100 {
		return cluster.Worker{}, fmt.Errorf(
			"%w: GPU utilization %.2f is outside 0-100",
			ErrInvalidDCGMSample,
			sample.GPUUtilizationPercent,
		)
	}

	if sample.FramebufferTotalMB == 0 {
		return cluster.Worker{}, fmt.Errorf(
			"%w: framebuffer total memory must be greater than zero",
			ErrInvalidDCGMSample,
		)
	}

	if sample.FramebufferFreeMB > sample.FramebufferTotalMB {
		return cluster.Worker{}, fmt.Errorf(
			"%w: framebuffer free memory %d exceeds total memory %d",
			ErrInvalidDCGMSample,
			sample.FramebufferFreeMB,
			sample.FramebufferTotalMB,
		)
	}

	if sample.FramebufferUsedMB > sample.FramebufferTotalMB {
		return cluster.Worker{}, fmt.Errorf(
			"%w: framebuffer used memory %d exceeds total memory %d",
			ErrInvalidDCGMSample,
			sample.FramebufferUsedMB,
			sample.FramebufferTotalMB,
		)
	}

	if sample.FramebufferFreeMB+sample.FramebufferUsedMB >
		sample.FramebufferTotalMB {
		return cluster.Worker{}, fmt.Errorf(
			"%w: framebuffer free and used memory exceed total memory",
			ErrInvalidDCGMSample,
		)
	}

	updated := worker

	updated.TotalMemoryMB = sample.FramebufferTotalMB
	updated.AvailableMemoryMB = sample.FramebufferFreeMB
	updated.ComputeUtilization = sample.GPUUtilizationPercent
	updated.MemoryUtilization =
		float64(sample.FramebufferUsedMB) /
			float64(sample.FramebufferTotalMB) *
			100

	updated.LastHeartbeat = sample.Timestamp

	if sample.NodeID != "" {
		updated.NodeID = sample.NodeID
	}

	if sample.GPUType != "" {
		updated.GPUType = sample.GPUType
	}

	return updated, nil
}
