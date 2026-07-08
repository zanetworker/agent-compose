package compose

import (
	"context"
	"sync"
)

type RunStore interface {
	Save(ctx context.Context, run *Run) error
	Get(ctx context.Context, agent string) (*Run, error)
	List(ctx context.Context) ([]Run, error)
	Delete(ctx context.Context, agent string) error
}

type MemoryStore struct {
	mu   sync.RWMutex
	runs map[string]*Run
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{runs: make(map[string]*Run)}
}

func (s *MemoryStore) Save(_ context.Context, run *Run) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runs[run.Agent] = run
	return nil
}

func (s *MemoryStore) Get(_ context.Context, agent string) (*Run, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	run, ok := s.runs[agent]
	if !ok {
		return nil, ErrNotFound
	}
	return run, nil
}

func (s *MemoryStore) List(_ context.Context) ([]Run, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	runs := make([]Run, 0, len(s.runs))
	for _, r := range s.runs {
		runs = append(runs, *r)
	}
	return runs, nil
}

func (s *MemoryStore) Delete(_ context.Context, agent string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.runs, agent)
	return nil
}
