package telemetry

import (
	"context"
	"time"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
)

type SnapshotProvider interface {
	Snapshot(ctx context.Context) (cluster.Snapshot, error)
}

type Service struct {
	provider  SnapshotProvider
	collector *Collector
	interval  time.Duration
}

func NewService(
	provider SnapshotProvider,
	collector *Collector,
	interval time.Duration,
) *Service {
	return &Service{
		provider:  provider,
		collector: collector,
		interval:  interval,
	}
}

func (s *Service) Run(ctx context.Context) error {
	if err := s.collect(ctx); err != nil {
		return err
	}

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case timestamp := <-ticker.C:
			if err := s.collectAt(ctx, timestamp); err != nil {
				return err
			}
		}
	}
}

func (s *Service) collect(ctx context.Context) error {
	return s.collectAt(ctx, time.Now())
}

func (s *Service) collectAt(
	ctx context.Context,
	timestamp time.Time,
) error {
	snapshot, err := s.provider.Snapshot(ctx)
	if err != nil {
		return err
	}

	return s.collector.RecordSnapshot(
		ctx,
		snapshot,
		timestamp,
	)
}
