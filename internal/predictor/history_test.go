package predictor

import (
	"testing"
	"time"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/telemetry"
)

func TestHistoryStoreKeepsBoundedHistory(t *testing.T) {
	store := NewHistoryStore(3)

	for i := 0; i < 5; i++ {
		store.Add(telemetry.Record{
			WorkerID:  "worker-1",
			Timestamp: time.Unix(int64(i), 0),
		})
	}

	records := store.Recent("worker-1", 10)

	if len(records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(records))
	}

	if records[0].Timestamp.Unix() != 2 {
		t.Fatalf("expected oldest retained timestamp 2, got %d", records[0].Timestamp.Unix())
	}
}

func TestHistoryStoreDeduplicatesTimestamp(t *testing.T) {
	store := NewHistoryStore(5)
	timestamp := time.Unix(100, 0)

	store.Add(telemetry.Record{
		WorkerID:   "worker-1",
		Timestamp:  timestamp,
		WorkloadID: "workload-1",
	})

	store.Add(telemetry.Record{
		WorkerID:   "worker-1",
		Timestamp:  timestamp,
		WorkloadID: "workload-2",
	})

	records := store.Recent("worker-1", 5)

	if len(records) != 1 {
		t.Fatalf("expected duplicate timestamp to be ignored, got %d records", len(records))
	}
}

func TestHistoryStoreSeparatesWorkers(t *testing.T) {
	store := NewHistoryStore(5)

	store.Add(telemetry.Record{WorkerID: "worker-1", Timestamp: time.Unix(1, 0)})
	store.Add(telemetry.Record{WorkerID: "worker-2", Timestamp: time.Unix(2, 0)})

	if len(store.Recent("worker-1", 5)) != 1 {
		t.Fatal("expected one worker-1 record")
	}

	if len(store.Recent("worker-2", 5)) != 1 {
		t.Fatal("expected one worker-2 record")
	}
}

func TestHistoryStoreReturnsDefensiveCopy(t *testing.T) {
	store := NewHistoryStore(5)

	store.Add(telemetry.Record{
		WorkerID:          "worker-1",
		Timestamp:         time.Unix(1, 0),
		MemoryUtilization: 50,
	})

	records := store.Recent("worker-1", 1)
	records[0].MemoryUtilization = 99

	again := store.Recent("worker-1", 1)
	if again[0].MemoryUtilization != 50 {
		t.Fatalf("history was mutated through returned slice: %.2f", again[0].MemoryUtilization)
	}
}
