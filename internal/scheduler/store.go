package scheduler

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// SchedulerStore manages persistent storage of ScheduledJobs
type SchedulerStore interface {
	List() ([]ScheduledJob, error)
	Get(id string) (*ScheduledJob, error)
	Save(job *ScheduledJob) error
	Delete(id string) error
}

// FileStore persists scheduled jobs to a JSON file
type FileStore struct {
	mu   sync.RWMutex
	path string
	jobs map[string]*ScheduledJob
}

// NewFileStore creates a FileStore, loading existing jobs from the file
func NewFileStore(path string) (*FileStore, error) {
	s := &FileStore{
		path: path,
		jobs: make(map[string]*ScheduledJob),
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, fmt.Errorf("reading scheduler store: %w", err)
	}
	var list []*ScheduledJob
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("parsing scheduler store: %w", err)
	}
	for _, j := range list {
		s.jobs[j.ID] = j
	}
	return s, nil
}

func (s *FileStore) persist() error {
	list := make([]*ScheduledJob, 0, len(s.jobs))
	for _, j := range s.jobs {
		list = append(list, j)
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

func (s *FileStore) List() ([]ScheduledJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ScheduledJob, 0, len(s.jobs))
	for _, j := range s.jobs {
		out = append(out, *j)
	}
	return out, nil
}

func (s *FileStore) Get(id string) (*ScheduledJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	j, ok := s.jobs[id]
	if !ok {
		return nil, fmt.Errorf("scheduled job %s not found", id)
	}
	copy := *j
	return &copy, nil
}

func (s *FileStore) Save(job *ScheduledJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job.UpdatedAt = time.Now()
	copy := *job
	s.jobs[job.ID] = &copy
	return s.persist()
}

func (s *FileStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.jobs, id)
	return s.persist()
}
