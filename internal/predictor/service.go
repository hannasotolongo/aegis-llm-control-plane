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
	predictor        *TrendPredictor
	history          *HistoryStore
	results          *ResultStore
	interval         time.Duration
	threshold        float64
	processedRecords int
}

func NewService(
	store TelemetryStore,
	predictor *TrendPredictor,
	results *ResultStore,
	interval time.Duration,
) *Service {
	return &Service{
		store:     store,
		predictor: predictor,
		history:   NewHistoryStore(16),
		results:   results,
		interval:  interval,
		threshold: DefaultConfidenceThreshold,
	}
}

func (s *Service) Run(ctx context.Context) error {
	if s.interval <= 0 {
		return errors.New("prediction interval must be positive")
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
		recent := s.history.Recent(workerID, 3)

		prediction, err := s.predictor.Predict(recent)
		if errors.Is(err, ErrInsufficientHistory) {
			continue
		}
		if err != nil {
			return err
		}

		decision := Decide(prediction, s.threshold)

		s.results.Set(Result{
			Prediction:  prediction,
			Decision:    decision,
			GeneratedAt: time.Now(),
		})

		log.Printf(
			"prediction worker=%s memory=%.2f compute=%.2f contention=%t confidence=%.2f mode=%s",
			workerID,
			prediction.PredictedMemoryUtilization,
			prediction.PredictedComputeUtilization,
			prediction.PredictedContention,
			prediction.Confidence,
			decision.Mode,
		)
	}

	return nil
}
