package predictor

import (
	"errors"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/telemetry"
)

var ErrInsufficientHistory = errors.New("insufficient telemetry history")

type Prediction struct {
	WorkerID                    string
	PredictedMemoryUtilization  float64
	PredictedComputeUtilization float64
	PredictedContention         bool
	Confidence                  float64
}

type TrendPredictor struct {
	MemoryThreshold  float64
	ComputeThreshold float64
}

func NewTrendPredictor() *TrendPredictor {
	return &TrendPredictor{
		MemoryThreshold:  telemetry.DefaultMemoryPressureThreshold,
		ComputeThreshold: telemetry.DefaultComputePressureThreshold,
	}
}

func (p *TrendPredictor) Predict(records []telemetry.Record) (Prediction, error) {
	if len(records) < 3 {
		return Prediction{}, ErrInsufficientHistory
	}

	first := records[len(records)-3]
	second := records[len(records)-2]
	latest := records[len(records)-1]

	memoryDelta1 := second.MemoryUtilization - first.MemoryUtilization
	memoryDelta2 := latest.MemoryUtilization - second.MemoryUtilization
	computeDelta1 := second.ComputeUtilization - first.ComputeUtilization
	computeDelta2 := latest.ComputeUtilization - second.ComputeUtilization

	memoryTrend := (memoryDelta1 + memoryDelta2) / 2
	computeTrend := (computeDelta1 + computeDelta2) / 2

	predictedMemory := clamp(latest.MemoryUtilization+memoryTrend, 0, 100)
	predictedCompute := clamp(latest.ComputeUtilization+computeTrend, 0, 100)

	confidence := trendConfidence(
		memoryDelta1,
		memoryDelta2,
		computeDelta1,
		computeDelta2,
	)

	return Prediction{
		WorkerID:                    latest.WorkerID,
		PredictedMemoryUtilization:  predictedMemory,
		PredictedComputeUtilization: predictedCompute,
		PredictedContention: predictedMemory >= p.MemoryThreshold ||
			predictedCompute >= p.ComputeThreshold,
		Confidence: confidence,
	}, nil
}

func trendConfidence(memoryDelta1, memoryDelta2, computeDelta1, computeDelta2 float64) float64 {
	memoryDifference := abs(memoryDelta2 - memoryDelta1)
	computeDifference := abs(computeDelta2 - computeDelta1)

	averageDifference := (memoryDifference + computeDifference) / 2

	confidence := 1 - averageDifference/100
	return clamp(confidence, 0, 1)
}

func clamp(value, minimum, maximum float64) float64 {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

func abs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
