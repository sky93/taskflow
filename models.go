package taskflow

import (
	"encoding/json"
	"time"
)

// JobStatus enumerates the possible states of a job.
type JobStatus string

const (
	JobPending    JobStatus = "PENDING"
	JobInProgress JobStatus = "IN_PROGRESS"
	JobCompleted  JobStatus = "COMPLETED"
	JobFailed     JobStatus = "FAILED"
)

// Operation is a type for your job "name" or "action" (e.g., "ADD_CUSTOMER").
type Operation string

// JobRecord corresponds to one row in the jobs table.
type JobRecord struct {
	ID          uint64
	Operation   Operation
	Status      JobStatus
	payload     []byte
	Output      any
	LockedBy    *string
	LockedUntil *time.Time
	RetryCount  uint
	AvailableAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (jr *JobRecord) GetPayload(input any) error {
	return json.Unmarshal(jr.payload, &input)
}
