package scheduler

import (
	"context"
	"errors"
	"time"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
	"github.com/hannasotolongo/aegis-llm-control-plane/internal/risk"
)

type Service struct {
	store       cluster.StateStore
	predictions PredictionProvider
	evaluator   *risk.Evaluator
}

func NewService(
	store cluster.StateStore,
) *Service {
	return &Service{
		store: store,
	}
}

func NewPredictiveService(
	store cluster.StateStore,
	predictions PredictionProvider,
) *Service {
	return &Service{
		store:       store,
		predictions: predictions,
	}
}

func NewRiskAwareService(
	store cluster.StateStore,
	predictions PredictionProvider,
	evaluator *risk.Evaluator,
) *Service {
	return &Service{
		store:       store,
		predictions: predictions,
		evaluator:   evaluator,
	}
}

func (s *Service) SchedulePending(
	ctx context.Context,
) ([]cluster.Workload, error) {
	snapshot, err := s.store.Snapshot(ctx)
	if err != nil {
		return nil, err
	}

	pending := make(
		[]cluster.Workload,
		0,
	)

	for _, workload := range snapshot.Workloads {
		if workload.State == cluster.WorkloadPending ||
			workload.State == cluster.WorkloadQueued {
			pending = append(
				pending,
				workload,
			)
		}
	}

	ordered := OrderWorkloadsWithAging(
		pending,
		time.Now(),
	)

	placed := make(
		[]cluster.Workload,
		0,
		len(ordered),
	)

	for _, workload := range ordered {
		result, err := s.placeWorkload(
			ctx,
			workload.ID,
		)
		if err != nil {
			if errors.Is(
				err,
				ErrNoEligibleWorker,
			) {
				continue
			}

			return placed, err
		}

		placed = append(
			placed,
			result,
		)
	}

	return placed, nil
}

func (s *Service) placeWorkload(
	ctx context.Context,
	workloadID string,
) (cluster.Workload, error) {
	if s.evaluator != nil {
		return PlaceWorkloadRiskAware(
			ctx,
			s.store,
			workloadID,
			s.predictions,
			s.evaluator,
		)
	}

	if s.predictions != nil {
		return PlaceWorkloadPredictive(
			ctx,
			s.store,
			workloadID,
			s.predictions,
		)
	}

	return PlaceWorkload(
		ctx,
		s.store,
		workloadID,
	)
}

func (s *Service) Run(
	ctx context.Context,
	interval time.Duration,
) error {
	if interval <= 0 {
		return errors.New(
			"scheduler interval must be positive",
		)
	}

	if _, err := s.SchedulePending(ctx); err != nil {
		return err
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			if _, err := s.SchedulePending(
				ctx,
			); err != nil {
				return err
			}
		}
	}
}
