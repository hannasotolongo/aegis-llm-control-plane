package predictor

import (
	"errors"
	"time"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/telemetry"
)

var (
	ErrInsufficientHistory = errors.New(
		"insufficient telemetry history",
	)

	ErrInvalidTelemetryInterval = errors.New(
		"telemetry timestamps must be strictly increasing",
	)
)

type TrendPredictor struct {
	MemoryThreshold  float64
	ComputeThreshold float64
	estimator        UncertaintyEstimator
}

var _ Forecaster = (*TrendPredictor)(nil)

func NewTrendPredictor() *TrendPredictor {
	return &TrendPredictor{
		MemoryThreshold:  telemetry.DefaultMemoryPressureThreshold,
		ComputeThreshold: telemetry.DefaultComputePressureThreshold,
		estimator:        NewMeanAbsoluteErrorEstimator(),
	}
}

func (p *TrendPredictor) Forecast(
	records []telemetry.Record,
	horizon time.Duration,
) (Forecast, error) {
	if len(records) < 3 {
		return Forecast{}, ErrInsufficientHistory
	}

	forecast, err := p.forecastAtHorizon(
		records[len(records)-3:],
		horizon,
	)
	if err != nil {
		return Forecast{}, err
	}

	forecast.Uncertainty = p.estimateUncertainty(
		records,
	)

	return forecast, nil
}

func (p *TrendPredictor) forecastAtHorizon(
	records []telemetry.Record,
	horizon time.Duration,
) (Forecast, error) {
	first := records[len(records)-3]
	second := records[len(records)-2]
	latest := records[len(records)-1]

	firstInterval :=
		second.Timestamp.Sub(first.Timestamp)

	secondInterval :=
		latest.Timestamp.Sub(second.Timestamp)

	if firstInterval <= 0 ||
		secondInterval <= 0 {
		return Forecast{}, ErrInvalidTelemetryInterval
	}

	memoryRate1 :=
		(second.MemoryUtilization -
			first.MemoryUtilization) /
			firstInterval.Seconds()

	memoryRate2 :=
		(latest.MemoryUtilization -
			second.MemoryUtilization) /
			secondInterval.Seconds()

	computeRate1 :=
		(second.ComputeUtilization -
			first.ComputeUtilization) /
			firstInterval.Seconds()

	computeRate2 :=
		(latest.ComputeUtilization -
			second.ComputeUtilization) /
			secondInterval.Seconds()

	memoryRate :=
		(memoryRate1 + memoryRate2) / 2

	computeRate :=
		(computeRate1 + computeRate2) / 2

	horizonSeconds := horizon.Seconds()

	predictedMemory := clamp(
		latest.MemoryUtilization+
			memoryRate*horizonSeconds,
		0,
		100,
	)

	predictedCompute := clamp(
		latest.ComputeUtilization+
			computeRate*horizonSeconds,
		0,
		100,
	)

	return Forecast{
		WorkerID:                    latest.WorkerID,
		Horizon:                     horizon,
		PredictedMemoryUtilization:  predictedMemory,
		PredictedComputeUtilization: predictedCompute,
		PredictedContention: predictedMemory >= p.MemoryThreshold ||
			predictedCompute >= p.ComputeThreshold,
	}, nil
}

func (p *TrendPredictor) forecastNext(
	records []telemetry.Record,
) Forecast {
	first := records[len(records)-3]
	second := records[len(records)-2]
	latest := records[len(records)-1]

	memoryDelta1 :=
		second.MemoryUtilization -
			first.MemoryUtilization

	memoryDelta2 :=
		latest.MemoryUtilization -
			second.MemoryUtilization

	computeDelta1 :=
		second.ComputeUtilization -
			first.ComputeUtilization

	computeDelta2 :=
		latest.ComputeUtilization -
			second.ComputeUtilization

	memoryTrend :=
		(memoryDelta1 + memoryDelta2) / 2

	computeTrend :=
		(computeDelta1 + computeDelta2) / 2

	predictedMemory := clamp(
		latest.MemoryUtilization+memoryTrend,
		0,
		100,
	)

	predictedCompute := clamp(
		latest.ComputeUtilization+computeTrend,
		0,
		100,
	)

	return Forecast{
		WorkerID:                    latest.WorkerID,
		PredictedMemoryUtilization:  predictedMemory,
		PredictedComputeUtilization: predictedCompute,
		PredictedContention: predictedMemory >= p.MemoryThreshold ||
			predictedCompute >= p.ComputeThreshold,
	}
}

func (p *TrendPredictor) estimateUncertainty(
	records []telemetry.Record,
) Uncertainty {
	if len(records) < 4 {
		return Uncertainty{}
	}

	samples := ErrorSamples{
		ActualMemory:     make([]float64, 0, len(records)-3),
		PredictedMemory:  make([]float64, 0, len(records)-3),
		ActualCompute:    make([]float64, 0, len(records)-3),
		PredictedCompute: make([]float64, 0, len(records)-3),
	}

	for i := 3; i < len(records); i++ {
		history := records[i-3 : i]

		forecast := p.forecastNext(
			history,
		)

		actual := records[i]

		samples.ActualMemory = append(
			samples.ActualMemory,
			actual.MemoryUtilization,
		)

		samples.PredictedMemory = append(
			samples.PredictedMemory,
			forecast.PredictedMemoryUtilization,
		)

		samples.ActualCompute = append(
			samples.ActualCompute,
			actual.ComputeUtilization,
		)

		samples.PredictedCompute = append(
			samples.PredictedCompute,
			forecast.PredictedComputeUtilization,
		)
	}

	uncertainty, err := p.estimator.Estimate(
		samples,
	)
	if err != nil {
		return Uncertainty{}
	}

	return uncertainty
}

func clamp(
	value float64,
	minimum float64,
	maximum float64,
) float64 {
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
