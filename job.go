package taskflow

import "time"

// AdvancedJob Job is the interface each job must implement.
type AdvancedJob interface {
	Run(jr JobRecord) (any, error)
	RetryCount() uint           // returns how many times we allow this job to fail
	BackoffTime() time.Duration // how long to wait after a failure
	JobTimeout() time.Duration  // how long this job may run before timing out
}
