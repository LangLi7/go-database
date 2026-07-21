package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"go-database/internal/transfer"
)

// Scheduler manages recurring transfer jobs
type Scheduler struct {
	engine  transfer.TransferEngine
	store   SchedulerStore
	mu      sync.RWMutex
	cancel  context.CancelFunc
	running bool
}

// New creates a new scheduler
func New(engine transfer.TransferEngine, store SchedulerStore) *Scheduler {
	return &Scheduler{
		engine: engine,
		store:  store,
	}
}

// Start begins the scheduler loop
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	ctx, s.cancel = context.WithCancel(ctx)
	s.running = true
	s.mu.Unlock()

	go s.loop(ctx)
}

// Stop stops the scheduler loop
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
	s.running = false
}

func (s *Scheduler) loop(ctx context.Context) {
	slog.Info("scheduler started")
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Check immediately on start
	s.check(ctx)

	for {
		select {
		case <-ticker.C:
			s.check(ctx)
		case <-ctx.Done():
			slog.Info("scheduler stopped")
			return
		}
	}
}

func (s *Scheduler) check(ctx context.Context) {
	jobs, err := s.store.List()
	if err != nil {
		slog.Error("scheduler list error", "err", err)
		return
	}

	now := time.Now()
	for _, j := range jobs {
		if !j.Enabled {
			continue
		}
		if j.NextRunAt == nil || j.NextRunAt.Before(now) {
			s.execute(ctx, &j)
		}
	}
}

func (s *Scheduler) execute(ctx context.Context, job *ScheduledJob) {
	slog.Info("scheduled job firing", "id", job.ID, "name", job.Name)

	targetJob := &transfer.TransferJob{
		SourceConn: job.SourceConn,
		TargetConn: job.TargetConn,
		Tables:     job.Tables,
		DryRun:     false,
		BatchSize:  job.BatchSize,
		OnConflict: job.OnConflict,
	}

	if err := s.engine.Start(ctx, targetJob); err != nil {
		slog.Error("scheduled job start error", "id", job.ID, "err", err)
		s.recordRun(job.ID, "failed")
		return
	}

	// Wait for completion
	status := "success"
	for {
		st, err := s.engine.Status(targetJob.ID)
		if err != nil {
			status = "failed"
			break
		}
		if st.Status == "done" || st.Status == "failed" {
			if st.Status == "failed" {
				status = "failed"
			}
			break
		}
		select {
		case <-time.After(2 * time.Second):
		case <-ctx.Done():
			return
		}
	}

	s.recordRun(job.ID, status)
}

func (s *Scheduler) recordRun(jobID, status string) {
	j, err := s.store.Get(jobID)
	if err != nil {
		return
	}
	now := time.Now()
	j.LastRunAt = &now
	j.LastStatus = status
	next, err := nextTime(j.CronExpr, now)
	if err == nil {
		j.NextRunAt = &next
	}
	s.store.Save(j)
}

// EnsureJob creates or updates a scheduled job and recalculates NextRunAt
func (s *Scheduler) EnsureJob(job *ScheduledJob) error {
	if job.CronExpr != "" {
		now := time.Now()
		next, err := nextTime(job.CronExpr, now)
		if err != nil {
			return fmt.Errorf("invalid cron expression: %w", err)
		}
		job.NextRunAt = &next
	}
	return s.store.Save(job)
}

// GenID generates a unique ID for a scheduled job
func GenID() string {
	return fmt.Sprintf("sched-%d", time.Now().UnixNano())
}
