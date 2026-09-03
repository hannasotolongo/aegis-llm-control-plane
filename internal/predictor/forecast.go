package predictor

import (
	"time"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/telemetry"
)

type Forecast struct {
	WorkerID                    string
	Horizon                     time.Duration
	PredictedMemoryUtilization  float64
	PredictedComputeUtilization float64
	PredictedContention         bool
	Uncertainty                 Uncertainty
}

type Forecaster interface {
	Forecast(
		records []telemetry.Record,
		horizon time.Duration,
	) (Forecast, error)
}
