package predictor

import (
	"context"
	"testing"
	"time"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/telemetry"
)

type fakeTelemetryStore struct {
	records []telemetry.Record
}

func (s *fakeTelemetryStore) List(ctx context.Context) ([]telemetry.Record, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	result := make([]telemetry.Record, len(s.records))
	copy(result, s.records)
	return result, nil
}

func TestServiceProcessesOnlyNewTelemetry(t *testing.T) {
	base := time.Unix(100, 0)

	store := &fakeTelemetryStore{
		records: []telemetry.Record{
			{WorkerID: "worker-1", Timestamp: base, MemoryUtilization: 40, ComputeUtilization: 30},
			{WorkerID: "worker-1", Timestamp: base.Add(time.Second), MemoryUtilization: 50, ComputeUtilization: 40},
			{WorkerID: "worker-1", Timestamp: base.Add(2 * time.Second), MemoryUtilization: 60, ComputeUtilization: 50},
		},
	}

	results := NewResultStore()
	service := NewService(store, NewTrendPredictor(), results, time.Second)

	if err := service.predict(context.Background()); err != nil {
		t.Fatalf("first predict failed: %v", err)
	}

	if service.processedRecords != 3 {
		t.Fatalf("expected 3 processed records, got %d", service.processedRecords)
	}

	result, ok := results.Get("worker-1")
	if !ok {
		t.Fatal("expected prediction result for worker-1")
	}

	if result.Prediction.WorkerID != "worker-1" {
		t.Fatalf("unexpected prediction worker %q", result.Prediction.WorkerID)
	}

	before := service.history.Recent("worker-1", 16)
	if len(before) != 3 {
		t.Fatalf("expected 3 history records, got %d", len(before))
	}

	store.records = append(store.records, telemetry.Record{
		WorkerID:           "worker-1",
		Timestamp:          base.Add(3 * time.Second),
		MemoryUtilization:  70,
		ComputeUtilization: 60,
	})

	if err := service.predict(context.Background()); err != nil {
		t.Fatalf("second predict failed: %v", err)
	}

	if service.processedRecords != 4 {
		t.Fatalf("expected 4 processed records, got %d", service.processedRecords)
	}

	after := service.history.Recent("worker-1", 16)
	if len(after) != 4 {
		t.Fatalf("expected 4 history records, got %d", len(after))
	}

	if !after[len(after)-1].Timestamp.Equal(base.Add(3 * time.Second)) {
		t.Fatal("expected newest telemetry record to be retained")
	}
}
