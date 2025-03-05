package taskflow

import (
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
FROM card.jobs
WHERE 
  (status = 'PENDING' OR status = 'FAILED')
  AND (locked_until IS NULL OR locked_until < NOW())
  AND retry_count < ?
  AND available_at <= NOW()
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
		&rec.Payload,
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
UPDATE card.jobs
SET 
  status = 'IN_PROGRESS',
  locked_by = ?,
  locked_until = ?,
  updated_at = ?
WHERE id = ?
`
	_, err := tx.Exec(stmt,
		workerID,
		lockUntil,
		time.Now().UTC().Round(time.Second),
		jobID,
	)
	return err
}

func finishJob(db *sql.DB, jobID uint64, finalStatus JobStatus, output *string, incrementRetry bool, availableAt *time.Time) error {
	outputJson, _ := json.Marshal(output)

	setClauses := []string{
		"status = ?",
		"output = ?",
		"updated_at = ?",
		"locked_by = NULL",
		"locked_until = NULL",
	}
	args := []interface{}{
		finalStatus,
		outputJson,
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

	query := fmt.Sprintf("UPDATE card.jobs SET %s WHERE id = ?", strings.Join(setClauses, ", "))
	_, err := db.Exec(query, args...)
	return err
}
