package predictor

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/telemetry"
)

type TelemetryStore interface {
	List(ctx context.Context) ([]telemetry.Record, error)
}

type Service struct {
	store            TelemetryStore
	forecaster       Forecaster
	history          *HistoryStore
	results          *ResultStore
	interval         time.Duration
	horizon          time.Duration
	processedRecords int
}

func NewService(
	store TelemetryStore,
	forecaster Forecaster,
	results *ResultStore,
	interval time.Duration,
	horizon time.Duration,
) *Service {
	return &Service{
		store:      store,
		forecaster: forecaster,
		history:    NewHistoryStore(16),
		results:    results,
		interval:   interval,
		horizon:    horizon,
	}
}

func (s *Service) Run(ctx context.Context) error {
	if s.interval <= 0 {
		return errors.New("prediction interval must be positive")
	}

	if s.horizon <= 0 {
		return errors.New("forecast horizon must be positive")
	}

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			if err := s.predict(ctx); err != nil {
				return err
			}
		}
	}
}

func (s *Service) predict(ctx context.Context) error {
	records, err := s.store.List(ctx)
	if err != nil {
		return err
	}

	if len(records) < s.processedRecords {
		s.history = NewHistoryStore(16)
		s.processedRecords = 0
	}

	for _, record := range records[s.processedRecords:] {
		s.history.Add(record)
	}

	s.processedRecords = len(records)

	for _, workerID := range s.history.WorkerIDs() {
		recent := s.history.Recent(workerID, 16)

		forecast, err := s.forecaster.Forecast(
			recent,
			s.horizon,
		)

		if errors.Is(err, ErrInsufficientHistory) {
			continue
		}

		if err != nil {
			return err
		}

		s.results.Set(Result{
			Forecast:    forecast,
			GeneratedAt: time.Now(),
		})

		log.Printf(
			"forecast worker=%s horizon=%s memory=%.2f compute=%.2f contention=%t",
			workerID,
			forecast.Horizon,
			forecast.PredictedMemoryUtilization,
			forecast.PredictedComputeUtilization,
			forecast.PredictedContention,
		)
	}

	return nil
}
