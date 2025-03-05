package main

import (
	"context"
	"database/sql"
	"fmt"
	"***REMOVED***/taskflow"
	"time"

	// Suppose your custom job is here:
	"***REMOVED***/taskflow/test/jobs"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// 1) Open *sql.DB
	dsn := "root:mju7&UJM@tcp(127.0.0.1:3306)/card?charset=utf8mb4&parseTime=True"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}

	// Optionally, verify connection
	if err = db.Ping(); err != nil {
		panic(err)
	}
	fmt.Println("Connected to database.")

	// 2) Create the taskflow config
	cfg := &taskflow.Config{
		DB:           db, // *sql.DB
		RetryCount:   5,
		BackoffTime:  30 * time.Second,
		PollInterval: 5 * time.Second,
		JobTimeout:   10 * time.Second, // for example
		InfoLog: func(ev taskflow.LogEvent) {
			fmt.Println("info ", ev)
		},
		ErrorLog: func(ev taskflow.LogEvent) {
			fmt.Println("err ", ev)
		},
	}

	// 3) Register your job handlers
	taskflow.RegisterHandler("test", taskflow.MakeHandler(addCustomer.New))
	// If you have others: OperationIssueCard, etc., do similarly.

	// 4) Start the workers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Let's run 2 worker goroutines
	flow := taskflow.New(cfg)
	flow.StartWorkers(ctx, 2)

	// Keep the main process alive until user kills it or context times out
	select {}
}
