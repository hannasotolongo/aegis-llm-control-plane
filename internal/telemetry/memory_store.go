package telemetry

import (
	"context"
	"sync"
)

type InMemoryStore struct {
	mu      sync.RWMutex
	records []Record
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		records: make([]Record, 0),
	}
}

func (s *InMemoryStore) Append(
	ctx context.Context,
	record Record,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.records = append(s.records, record)

	return nil
}

func (s *InMemoryStore) List(
	ctx context.Context,
) ([]Record, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	records := make([]Record, len(s.records))
	copy(records, s.records)

	return records, nil
}

var _ Store = (*InMemoryStore)(nil)
