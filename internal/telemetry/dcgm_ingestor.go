package telemetry

import (
	"context"
	"fmt"
	"time"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
)

type DCGMIngestor struct {
	store  cluster.StateStore
	maxAge time.Duration
}

func NewDCGMIngestor(
	store cluster.StateStore,
	maxAge time.Duration,
) *DCGMIngestor {
	if maxAge <= 0 {
		maxAge = DefaultDCGMMaxAge
	}

	return &DCGMIngestor{
		store:  store,
		maxAge: maxAge,
	}
}

func (i *DCGMIngestor) Ingest(
	ctx context.Context,
	sample DCGMSample,
	now time.Time,
) error {
	worker, err := i.store.GetWorker(
		ctx,
		sample.WorkerID,
	)
	if err != nil {
		return fmt.Errorf(
			"get worker %q: %w",
			sample.WorkerID,
			err,
		)
	}

	updated, err := ApplyDCGMSample(
		worker,
		sample,
		now,
		i.maxAge,
	)
	if err != nil {
		return err
	}

	if err := i.store.UpdateWorker(
		ctx,
		updated,
	); err != nil {
		return fmt.Errorf(
			"update worker %q from DCGM telemetry: %w",
			sample.WorkerID,
			err,
		)
	}

	return nil
}
