package taskflow

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Manager struct {
	cfg              *Config
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	handlers         map[Operation]JobHandler
	advancedHandlers map[Operation]AdvancedJobConstructor
	wakeup           chan struct{}
}

// startWorkersInternal is basically your old StartWorkers logic
func startWorkersInternal(ctx context.Context, count int, cfg *Config, handlers map[Operation]JobHandler, advHandlers map[Operation]AdvancedJobConstructor) *Manager {
	mgrCtx, cancel := context.WithCancel(ctx)
	mgr := &Manager{
		cfg:              cfg,
		ctx:              mgrCtx,
		cancel:           cancel,
		handlers:         handlers,
		advancedHandlers: advHandlers,
		wakeup:           make(chan struct{}, count),
	}

	cfg.logInfo(LogEvent{
		Message: fmt.Sprintf("Starting %d workers...", count),
	})

	for i := 0; i < count; i++ {
		w := &Worker{
			id:      fmt.Sprintf("worker-%d", i),
			cfg:     cfg,
			manager: mgr,
		}
		mgr.wg.Add(1)
		go func(worker *Worker) {
			defer mgr.wg.Done()
			worker.Run(mgr.ctx)
		}(w)
	}

	return mgr
}

// Shutdown attempts a graceful shutdown: cancel context, wait for workers up to 'timeout'.
func (m *Manager) Shutdown(timeout time.Duration) {
	m.cfg.logInfo(LogEvent{Message: "Shutdown requested. Stopping workers..."})
	m.cancel()

	doneCh := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(doneCh)
	}()

	select {
	case <-doneCh:
		m.cfg.logInfo(LogEvent{Message: "All workers exited cleanly."})
	case <-time.After(timeout):
		m.cfg.logError(LogEvent{
			Message: fmt.Sprintf("Shutdown timed out after %v. Some workers may still be running.", timeout),
		})
	}
}
