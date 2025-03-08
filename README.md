<div align="center">
  <img src="logo.png" alt="tracesight" width="200px">
</div>

# ⏳taskflow
---
[![Go Reference](https://pkg.go.dev/badge/github.com/sky93/taskflow.svg)](https://pkg.go.dev/github.com/sky93/taskflow)
[![Go Report Card](https://goreportcard.com/badge/github.com/sky93/taskflow)](https://goreportcard.com/report/github.com/sky93/taskflow)

**taskflow** is a lightweight Go library for running background jobs from a MySQL queue table. It handles:

- Fetching jobs from the database
- Locking and retrying failed jobs
- Creating new jobs programmatically
- Running custom job logic with optional timeouts
- Structured logging via user-defined callbacks
- Graceful shutdown of worker pools

## Installation

```bash
go get github.com/sky93/taskflow
```

---

## Database Schema

You must create or already have a `card.jobs` table. For example:

```sql
CREATE TABLE IF NOT EXISTS card.jobs (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  operation VARCHAR(50) NOT NULL,
  status ENUM('PENDING','IN_PROGRESS','COMPLETED','FAILED') NOT NULL DEFAULT 'PENDING',
  payload TEXT NULL,
  output TEXT NULL,
  locked_by VARCHAR(50) NULL,
  locked_until DATETIME NULL,
  retry_count INT UNSIGNED NOT NULL DEFAULT 0,
  available_at DATETIME NOT NULL DEFAULT '1970-01-01 00:00:00',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);
```

---

## Quick Start Example

Below is a complete example demonstrating how to:

1. **Initialize** a `*sql.DB`
2. **Create** a `Config` and pass it to `taskflow.New(...)`
3. **Register** a custom job handler
4. **Create** a job
5. **Start** workers
6. **Shut down** gracefully

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "time"

    "github.com/sky93/taskflow"
    _ "github.com/go-sql-driver/mysql"
)

// Our "Hello" job
type HelloJob struct {
    name   string
    output string
}

func NewHelloJob(payload *string) (taskflow.Job, error) {
    if payload == nil || *payload == "" {
        return nil, fmt.Errorf("invalid or empty payload")
    }
    return &HelloJob{name: *payload}, nil
}

func (j *HelloJob) Run() error {
    // Simulate some logic
    j.output = "Hello, " + j.name + "!"
    return nil
}

func (j *HelloJob) GetOutput() *string {
    if j.output == "" {
        return nil
    }
    return &j.output
}

func main() {
    // 1) Connect to the DB
    dsn := "root:password@tcp(127.0.0.1:3306)/card?parseTime=true"
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        panic(err)
    }
    if err := db.Ping(); err != nil {
        panic(err)
    }
    fmt.Println("Connected to database.")

    // 2) Create the taskflow config
    cfg := &taskflow.Config{
        DB:           db,
        RetryCount:   3,
        BackoffTime:  30 * time.Second,
        PollInterval: 5 * time.Second,
        JobTimeout:   10 * time.Second,

        // Optional structured logging
        InfoLog: func(ev taskflow.LogEvent) {
            fmt.Printf("[INFO] %s\n", ev.Message)
            if ev.Err != nil {
                fmt.Printf("       error: %v\n", ev.Err)
            }
        },
        ErrorLog: func(ev taskflow.LogEvent) {
            fmt.Printf("[ERROR] %s\n", ev.Message)
            if ev.Err != nil {
                fmt.Printf("        error: %v\n", ev.Err)
            }
            fmt.Println()
        },
    }

    // 3) Create an instance of TaskFlow
    flow := taskflow.New(cfg)

    // 4) Register our handler (globally for "HELLO")
    taskflow.RegisterHandler("HELLO", taskflow.MakeHandler(NewHelloJob))

    // 5) Create a new job
    jobID, err := flow.CreateJob("HELLO", `"Alice"`)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Created job ID %d\n", jobID)

    // 6) Start workers
    ctx := context.Background()
    flow.StartWorkers(ctx, 2)

    // Let it run for 30 seconds
    time.Sleep(30 * time.Second)

    // 7) Shutdown gracefully
    flow.Shutdown(10 * time.Second)
    fmt.Println("All done.")
}
```