package scheduler

import "time"

// ScheduledJob represents a recurring transfer job
type ScheduledJob struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	SourceConn string     `json:"source_conn"`
	TargetConn string     `json:"target_conn"`
	Tables     []string   `json:"tables"`
	OnConflict string     `json:"on_conflict"`
	BatchSize  int        `json:"batch_size"`
	CronExpr   string     `json:"cron_expr"`
	Enabled    bool       `json:"enabled"`
	LastRunAt  *time.Time `json:"last_run_at,omitempty"`
	NextRunAt  *time.Time `json:"next_run_at,omitempty"`
	LastStatus string     `json:"last_status,omitempty"` // "success" | "failed" | ""
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}
