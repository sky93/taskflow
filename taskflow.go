package taskflow

import (
	"context"
	"fmt"
	"time"
)

type TaskFlow struct {
	cfg *Config
	mgr *Manager // We set this once we start workers
}

func New(cfg Config) *TaskFlow {
	// Provide default log functions if the user didn't set them
	if cfg.InfoLog == nil {
		cfg.InfoLog = defaultInfoLog
	}
	if cfg.ErrorLog == nil {
		cfg.ErrorLog = defaultErrorLog
	}

	return &TaskFlow{
		cfg: &cfg,
	}
}

// CreateJob inserts a new job into the database
func (tf *TaskFlow) CreateJob(operation Operation, payload string) (uint64, error) {
	return createJob(tf.cfg, operation, payload)
}

// StartWorkers spawns `count` workers to process jobs using the current config.
// It returns immediately, but you can call Shutdown(...) later to stop them.
func (tf *TaskFlow) StartWorkers(ctx context.Context, count int) {
	if tf.mgr != nil {
		tf.cfg.logError(LogEvent{
			Message: "Workers already started on this TaskFlow instance.",
		})
		return
	}
	mgr := startWorkersInternal(ctx, count, tf.cfg)
	tf.mgr = mgr
}

// Shutdown gracefully stops all workers, waiting up to `timeout` for them to exit.
func (tf *TaskFlow) Shutdown(timeout time.Duration) {
	if tf.mgr == nil {
		tf.cfg.logInfo(LogEvent{
			Message: "No workers to shut down (did you call StartWorkers?).",
		})
		return
	}
	tf.mgr.Shutdown(timeout)
	tf.mgr = nil
	tf.cfg.logInfo(LogEvent{Message: "TaskFlow shutdown complete."})
}

func createJob(cfg *Config, operation Operation, payload string) (uint64, error) {
	query := `
INSERT INTO card.jobs 
(operation, status, payload, locked_by, locked_until, retry_count, available_at, created_at, updated_at)
VALUES (?, 'PENDING', ?, NULL, NULL, 0, NOW(), NOW(), NOW())
`
	res, err := cfg.DB.Exec(query, operation, payload)
	if err != nil {
		return 0, fmt.Errorf("failed to insert job: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get lastInsertId: %w", err)
	}
	return uint64(id), nil
}
