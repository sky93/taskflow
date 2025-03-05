package taskflow

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type WorkerStatus int

const (
	WorkerIdle WorkerStatus = iota
	WorkerBusy
	WorkerFailing
	WorkerExecFailed
)

type Worker struct {
	id     string
	status WorkerStatus
	cfg    *Config

	manager    *Manager
	currentJob *JobRecord
}

// Run keeps polling the DB for jobs until context is canceled.
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()

	w.cfg.logInfo(LogEvent{
		Message:  fmt.Sprintf("Worker %s started.", w.id),
		WorkerID: w.id,
	})

	for {
		select {
		case <-ctx.Done():
			w.cfg.logInfo(LogEvent{
				Message:  fmt.Sprintf("Worker %s context canceled, stopping.", w.id),
				WorkerID: w.id,
			})
			return

		case <-ticker.C:
			w.cfg.logInfo(LogEvent{
				Message:  fmt.Sprintf("Worker %s polling for jobs...", w.id),
				WorkerID: w.id,
			})
			w.fetchAndProcess(ctx)
		}
	}
}

func (w *Worker) fetchAndProcess(ctx context.Context) {
	w.status = WorkerIdle

	tx, err := w.cfg.DB.Begin()
	if err != nil {
		w.cfg.logError(LogEvent{
			Message:  fmt.Sprintf("Error starting TX for worker %s", w.id),
			WorkerID: w.id,
			Err:      err,
		})
		return
	}

	jobRec, err := getPendingJob(tx, w.cfg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_ = tx.Commit()
			return
		}
		_ = tx.Rollback()
		w.status = WorkerFailing
		w.cfg.logError(LogEvent{
			Message:  fmt.Sprintf("Error fetching job for worker %s", w.id),
			WorkerID: w.id,
			Err:      err,
		})
		return
	}

	lockUntil := time.Now().Add(2 * time.Minute)
	if err := assignJobToWorker(tx, jobRec.ID, w.id, lockUntil); err != nil {
		_ = tx.Rollback()
		w.cfg.logError(LogEvent{
			Message:  fmt.Sprintf("Error assigning job %d to worker %s", jobRec.ID, w.id),
			WorkerID: w.id,
			JobID:    &jobRec.ID,
			Err:      err,
		})
		return
	}
	if err := tx.Commit(); err != nil {
		w.cfg.logError(LogEvent{
			Message:  fmt.Sprintf("Error committing assignment for job %d (worker %s)", jobRec.ID, w.id),
			WorkerID: w.id,
			JobID:    &jobRec.ID,
			Err:      err,
		})
		return
	}

	w.currentJob = jobRec
	w.status = WorkerBusy

	start := time.Now()
	opStr := string(jobRec.Operation)
	w.cfg.logInfo(LogEvent{
		Message:   fmt.Sprintf("Processing job %d (op: %s)", jobRec.ID, opStr),
		WorkerID:  w.id,
		JobID:     &jobRec.ID,
		Operation: &opStr,
	})

	// Actually execute the job
	output, execErr := w.executeJob(ctx, jobRec)

	finalStatus := JobCompleted
	var nextAvailableAt *time.Time
	incrementRetry := false

	if execErr != nil {
		finalStatus = JobFailed
		w.status = WorkerExecFailed

		if jobRec.Status == JobFailed {
			incrementRetry = true
		}
		t := time.Now().Add(w.cfg.BackoffTime)
		nextAvailableAt = &t
	}

	if err := finishJob(w.cfg.DB, jobRec.ID, finalStatus, output, incrementRetry, nextAvailableAt); err != nil {
		w.cfg.logError(LogEvent{
			Message:   fmt.Sprintf("Error finishing job %d", jobRec.ID),
			WorkerID:  w.id,
			JobID:     &jobRec.ID,
			Operation: &opStr,
			Err:       err,
		})
	}

	elapsed := time.Since(start)
	if execErr != nil {
		w.cfg.logError(LogEvent{
			Message:   fmt.Sprintf("Job %d FAILED in %v", jobRec.ID, elapsed),
			WorkerID:  w.id,
			JobID:     &jobRec.ID,
			Operation: &opStr,
			Duration:  &elapsed,
			Err:       execErr,
		})
	} else {
		w.cfg.logInfo(LogEvent{
			Message:   fmt.Sprintf("Job %d COMPLETED in %v", jobRec.ID, elapsed),
			WorkerID:  w.id,
			JobID:     &jobRec.ID,
			Operation: &opStr,
			Duration:  &elapsed,
		})
		w.status = WorkerIdle
	}

	w.currentJob = nil
}

// executeJob calls the appropriate handler for the job’s operation, optionally enforcing a timeout.
func (w *Worker) executeJob(ctx context.Context, jobRec *JobRecord) (*string, error) {
	handler, err := getHandler(jobRec.Operation)
	if err != nil {
		return nil, err
	}

	if w.cfg.JobTimeout <= 0 {
		return handler(jobRec.Payload)
	}

	// run in a sub-context with the user-defined timeout
	jobCtx, cancel := context.WithTimeout(ctx, w.cfg.JobTimeout)
	defer cancel()

	doneCh := make(chan struct{})
	var output *string
	var runErr error

	go func() {
		output, runErr = handler(jobRec.Payload)
		close(doneCh)
	}()

	select {
	case <-jobCtx.Done():
		msg := fmt.Sprintf("job timed out after %s", w.cfg.JobTimeout)
		return &msg, errors.New(msg)
	case <-doneCh:
		return output, runErr
	}
}
