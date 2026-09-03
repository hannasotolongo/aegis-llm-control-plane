package predictor

import (
	"sync"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/telemetry"
)

type HistoryStore struct {
	mu       sync.RWMutex
	capacity int
	records  map[string][]telemetry.Record
}

func NewHistoryStore(capacity int) *HistoryStore {
	if capacity < 3 {
		capacity = 3
	}

	return &HistoryStore{
		capacity: capacity,
		records:  make(map[string][]telemetry.Record),
	}
}

func (s *HistoryStore) Add(record telemetry.Record) {
	if record.WorkerID == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	history := s.records[record.WorkerID]

	if len(history) > 0 {
		latest := history[len(history)-1]
		if latest.Timestamp.Equal(record.Timestamp) {
			return
		}
	}

	history = append(history, record)

	if len(history) > s.capacity {
		history = history[len(history)-s.capacity:]
	}

	s.records[record.WorkerID] = history
}

func (s *HistoryStore) Recent(workerID string, count int) []telemetry.Record {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history := s.records[workerID]
	if len(history) == 0 || count <= 0 {
		return nil
	}

	if count > len(history) {
		count = len(history)
	}

	start := len(history) - count
	result := make([]telemetry.Record, count)
	copy(result, history[start:])

	return result
}

func (s *HistoryStore) WorkerIDs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]string, 0, len(s.records))
	for workerID := range s.records {
		ids = append(ids, workerID)
	}

	return ids
}
