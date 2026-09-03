package predictor

import (
	"sync"
	"time"
)

type Result struct {
	Forecast    Forecast
	GeneratedAt time.Time
}

type ResultStore struct {
	mu      sync.RWMutex
	results map[string]Result
}

func NewResultStore() *ResultStore {
	return &ResultStore{
		results: make(map[string]Result),
	}
}

func (s *ResultStore) Set(result Result) {
	if result.Forecast.WorkerID == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.results[result.Forecast.WorkerID] = result
}

func (s *ResultStore) Get(workerID string) (Result, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result, exists := s.results[workerID]
	return result, exists
}

func (s *ResultStore) Snapshot() map[string]Result {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make(map[string]Result, len(s.results))

	for workerID, result := range s.results {
		results[workerID] = result
	}

	return results
}
