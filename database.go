package taskflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// getPendingJob looks for a job with status in (PENDING, FAILED), not locked, retry < cfg.RetryCount, available now.
func getPendingJob(tx *sql.Tx, cfg *Config) (*JobRecord, error) {
	query := `
SELECT 
  id, 
  operation, 
  status, 
  payload, 
  output, 
  locked_by, 
  locked_until, 
  retry_count, 
  available_at,
  created_at,
  updated_at
FROM jobs
WHERE 
  (status = 'PENDING' OR status = 'FAILED')
  AND (locked_until IS NULL OR locked_until < NOW())
  AND retry_count < ?
  AND available_at <= NOW()
ORDER BY available_at
LIMIT 1
FOR UPDATE
`
	row := tx.QueryRow(query, cfg.RetryCount)
	var rec JobRecord
	var operationStr, statusStr string
	err := row.Scan(
		&rec.ID,
		&operationStr,
		&statusStr,
		&rec.payload,
		&rec.Output,
		&rec.LockedBy,
		&rec.LockedUntil,
		&rec.RetryCount,
		&rec.AvailableAt,
		&rec.CreatedAt,
		&rec.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// no job
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	rec.Operation = Operation(operationStr)
	rec.Status = JobStatus(statusStr)
	return &rec, nil
}

func assignJobToWorker(tx *sql.Tx, jobID uint64, workerID string, lockUntil time.Time) error {
	stmt := `
UPDATE jobs
SET 
  status = ?,
  locked_by = ?,
  locked_until = ?,
  updated_at = ?
WHERE id = ?
`
	_, err := tx.Exec(stmt,
		JobInProgress,
		workerID,
		lockUntil,
		time.Now().UTC().Round(time.Microsecond),
		jobID,
	)
	return err
}

func finishJob(db *sql.DB, jobID uint64, finalStatus JobStatus, output any, incrementRetry bool, availableAt *time.Time, errorOutput any) error {
	outputJson, err := json.Marshal(output)
	if err != nil {
		return err
	}
	outputQ := outputJson
	if outputJson == nil || len(outputJson) == 0 || string(outputJson) == "null" {
		outputQ = nil
	}

	errorOutputJson, err := json.Marshal(output)
	if err != nil {
		return err
	}
	errorOutputQ := errorOutputJson
	if errorOutputJson == nil || len(errorOutputJson) == 0 || string(errorOutputJson) == "null" {
		errorOutputJson = nil
	}

	setClauses := []string{
		"status = ?",
		"output = ?",
		"error_output = ?",
		"updated_at = ?",
		"locked_by = NULL",
		"locked_until = NULL",
	}
	args := []interface{}{
		finalStatus,
		outputQ,
		errorOutputQ,
		time.Now().UTC().Round(time.Microsecond),
	}

	if incrementRetry {
		setClauses = append(setClauses, "retry_count = retry_count + 1")
	}

	if availableAt != nil {
		setClauses = append(setClauses, "available_at = ?")
		args = append(args, *availableAt)
	}

	args = append(args, jobID)

	query := fmt.Sprintf("UPDATE jobs SET %s WHERE id = ?", strings.Join(setClauses, ", "))
	_, err = db.Exec(query, args...)
	return err
}

func createJob(ctx context.Context, tf *TaskFlow, operation Operation, payload any, executeAt time.Time) (int64, error) {
	plq, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}
	now := time.Now().Round(time.Microsecond)
	query := "INSERT INTO jobs (operation, status, payload, locked_by, locked_until, retry_count, available_at, created_at, updated_at) VALUES (?, ?, ?, NULL, NULL, 0, ?, ?, ?)"
	res, err := tf.cfg.DB.ExecContext(ctx, query, operation, JobPending, plq, executeAt, now, now)
	if err != nil {
		return 0, fmt.Errorf("failed to insert job: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get lastInsertId: %w", err)
	}
	return id, nil
}
