<div align="center">
  <img src="logo.png" alt="tracesight" width="200px">
</div>

# ⏳taskflow
[![Go Reference](https://pkg.go.dev/badge/github.com/sky93/taskflow.svg)](https://pkg.go.dev/github.com/sky93/taskflow)
[![Go Report Card](https://goreportcard.com/badge/github.com/sky93/taskflow)](https://goreportcard.com/report/github.com/sky93/taskflow)

**Taskflow** is a lightweight Go library for running background jobs out of a MySQL queue table. It handles:

- Fetching jobs from the database
- Locking and retrying failed jobs
- Creating new jobs programmatically
- Running custom job logic with optional timeouts
- Structured logging via user-defined callbacks
- Graceful shutdown of worker pools

---

## Table of Contents
1. [Installation](#installation)
2. [Database Schema](#database-schema)
3. [Quick Start Example](#quick-start-example)
4. [Contributing](#contributing)
5. [License](#license)

---

## Installation

```bash
go get github.com/sky93/taskflow
```

---

## Database Schema

Your database should contain a `jobs` table. For example:

```sql
CREATE TABLE IF NOT EXISTS jobs (
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

Below is a **complete**, minimal example showing:

1. Connecting to the database
2. Creating a `taskflow.Config` and a new `TaskFlow`
3. Registering a custom job handler
4. Starting workers
5. Creating a job
6. Shutting down gracefully

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "time"

    _ "github.com/go-sql-driver/mysql"
    "github.com/sky93/taskflow"
)

// MyPayload is the shape of the data we expect in the job payload.
type MyPayload struct {
    Greeting string
}

// HelloHandler processes jobs of type "HELLO".
func HelloHandler(jr taskflow.JobRecord) (any, error) {
    var payload MyPayload
    if err := jr.GetPayload(&payload); err != nil {
        return nil, err
    }

    // Here we just print the greeting; real logic can be anything.
    fmt.Println("Received greeting:", payload.Greeting)
    return nil, nil
}

func main() {
    ctx := context.Background()

    // 1) Connect to the DB
    dsn := "root:password@tcp(127.0.0.1:3306)/myDbName?parseTime=true"
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        panic(err)
    }
    if err := db.Ping(); err != nil {
        panic(err)
    }
    fmt.Println("Connected to database.")

    // 2) Create the taskflow config
    cfg := taskflow.Config{
        DB:           db,
        RetryCount:   3,
        BackoffTime:  30 * time.Second,
        PollInterval: 5 * time.Second,
        JobTimeout:   10 * time.Second,

        // Optional logging
        InfoLog: func(ev taskflow.LogEvent) {
            fmt.Printf("[INFO] %s\n", ev.Message)
        },
        ErrorLog: func(ev taskflow.LogEvent) {
            fmt.Printf("[ERROR] %s\n", ev.Message)
        },
    }

    // 3) Create an instance of TaskFlow
    flow := taskflow.New(cfg)

    // 4) Register our "HELLO" handler
    flow.RegisterHandler("HELLO", HelloHandler)

    // 5) Start workers (2 concurrent workers)
    flow.StartWorkers(ctx, 2)

    // Create a new "HELLO" job
    jobID, err := flow.CreateJob(ctx, "HELLO", MyPayload{Greeting: "Hello from TaskFlow!"}, time.Now())
    if err != nil {
        panic(err)
    }
    fmt.Printf("Created job ID %d\n", jobID)

    // Let it run for a few seconds
    time.Sleep(5 * time.Second)

    // 6) Shutdown gracefully
    flow.Shutdown(10 * time.Second)
    fmt.Println("All done.")
}
```

---

## Contributing

Contributions are welcome! Please follow these steps:

1. **Fork** the repository
2. Create a new **branch** for your feature or fix
3. Commit your changes, and **add tests** if possible
4. Submit a **pull request** and provide a clear description of your changes

---

## License

This project is licensed under the [MIT License](LICENSE).
