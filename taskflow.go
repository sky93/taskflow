package taskflow

import (
	"context"
	"sync"
	"time"
)

type TaskFlow struct {
	cfg       *Config
	mgr       *Manager // We set this once we start workers
	handlers  map[Operation]JobHandler
	handlerMu sync.RWMutex
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
		cfg:      &cfg,
		handlers: make(map[Operation]JobHandler),
	}
}

// CreateJob inserts a new job into the database
func (tf *TaskFlow) CreateJob(ctx context.Context, operation Operation, payload any, executeAt time.Time) (int64, error) {
	// Actually insert the job
	id, err := createJob(ctx, tf, operation, payload, executeAt)
	if err != nil {
		return 0, err
	}

	// If the job is ready now
	if executeAt.Before(time.Now()) {
		// Send a wakeup signal to let workers check immediately
		if tf.mgr != nil {
			select {
			case tf.mgr.wakeup <- struct{}{}:
				// wake worker
			default:
				// channel is full or no manager; ignore
			}
		}
	}
	return id, nil
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
	mgr := startWorkersInternal(ctx, count, tf.cfg, tf.handlers)
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
