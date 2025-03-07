package taskflow

import "time"

// Job is the interface each job must implement.
type Job interface {
	Run(jr JobRecord) (any, error)
	RetryCount() uint
	BackoffTime() time.Duration
	JobTimeout() time.Duration
}
