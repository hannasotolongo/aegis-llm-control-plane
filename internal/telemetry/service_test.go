package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
)

type fakeSnapshotProvider struct {
	snapshot cluster.Snapshot
}

func (f *fakeSnapshotProvider) Snapshot(
	ctx context.Context,
) (cluster.Snapshot, error) {
	return f.snapshot, nil
}

func TestServiceCollectsSnapshots(t *testing.T) {
	store := NewInMemoryStore()
	collector := NewCollector(store)

	provider := &fakeSnapshotProvider{
		snapshot: cluster.Snapshot{
			Workers: []cluster.Worker{
				{
					ID:                "worker-1",
					TotalMemoryMB:     80000,
					AvailableMemoryMB: 60000,
					State:             cluster.WorkerHealthy,
				},
			},
		},
	}

	service := NewService(
		provider,
		collector,
		10*time.Millisecond,
	)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		35*time.Millisecond,
	)
	defer cancel()

	err := service.Run(ctx)
	if err != nil && err != context.DeadlineExceeded {
		t.Fatalf("Run returned unexpected error: %v", err)
	}

	records, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if len(records) < 2 {
		t.Fatalf(
			"expected at least 2 telemetry records, got %d",
			len(records),
		)
	}

	for _, record := range records {
		if record.WorkerID != "worker-1" {
			t.Fatalf(
				"expected worker-1, got %q",
				record.WorkerID,
			)
		}
	}
}
