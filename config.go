package taskflow

import (
	"database/sql"
	"time"
)

// LogEvent captures information about a logging event.
type LogEvent struct {
	// A human-readable message about the event.
	Message string

	// The ID of the worker that triggered the log (if any).
	WorkerID string

	// The Job ID, if available.
	JobID *uint64

	// The operation name, if available.
	Operation *string

	// Any error associated with the event.
	Err error

	// How long the job or operation took, if relevant.
	Duration *time.Duration
}

// Config holds the settings and resources needed by the queue system.
type Config struct {
	// DB is the user-provided database connection where the jobs table is stored.
	DB *sql.DB

	// DbName is name of the database.
	DbName string

	// RetryCount is how many times we allow a job to fail before ignoring it.
	RetryCount uint

	// BackoffTime is how long we wait before letting a failed job become available again.
	BackoffTime time.Duration

	// PollInterval is how frequently workers check for new jobs.
	PollInterval time.Duration

	// JobTimeout is how long we allow an individual job to run before marking it as failed.
	// If zero, there is no enforced timeout.
	JobTimeout time.Duration

	// InfoLog is called for informational or success logs.
	// If nil, defaults to printing to stdout.
	InfoLog func(ev LogEvent)

	// ErrorLog is called for error logs.
	// If nil, defaults to printing to stderr (or stdout).
	ErrorLog func(ev LogEvent)
}
