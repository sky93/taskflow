package taskflow

import "fmt"

// JobHandler is a function that receives the payload string, constructs and runs the job,
// and returns an optional output string plus an error.
type JobHandler func(payload *string) (*string, error)

// registry of known handlers, keyed by Operation
var handlers = map[Operation]JobHandler{}

// RegisterHandler allows end users to associate an Operation with a JobHandler.
func RegisterHandler(op Operation, handler JobHandler) {
	handlers[op] = handler
}

// MakeHandler is a helper that wraps a "constructor" function (that returns a Job)
// into a JobHandler. Example usage:
//
//	taskflow.RegisterHandler(
//	    "HELLO",
//	    taskflow.MakeHandler(NewHelloJob),
//	)
func MakeHandler(constructor func(*string) (Job, error)) JobHandler {
	return func(payload *string) (*string, error) {
		j, err := constructor(payload)
		if err != nil {
			msg := err.Error()
			return &msg, err
		}
		if err := j.Run(); err != nil {
			return j.GetOutput(), err
		}
		return j.GetOutput(), nil
	}
}

// getHandler returns the JobHandler for the given operation or an error if not found.
func getHandler(op Operation) (JobHandler, error) {
	handler, ok := handlers[op]
	if !ok {
		msg := fmt.Sprintf("no handler registered for operation %s", op)
		return nil, fmt.Errorf(msg)
	}
	return handler, nil
}
