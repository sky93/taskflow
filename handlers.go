package taskflow

import "fmt"

// JobHandler is a function that receives the payload string, constructs and runs the job,
// and returns an optional output string plus an error.
type JobHandler func(jr JobRecord) (any, error)
type JobAdvancedHandler func(jr JobRecord) (any, error)

// RegisterHandler allows end users to associate an Operation with a JobHandler.
func (tf *TaskFlow) RegisterHandler(op Operation, handler JobHandler) {
	tf.handlerMu.Lock()
	tf.handlers[op] = makeHandler(handler)
	tf.handlerMu.Unlock()
}

func (tf *TaskFlow) RegisterAdvancedHandler(op Operation, handler JobHandler) {
	tf.handlerMu.Lock()
	tf.handlers[op] = makeHandler(handler)
	tf.handlerMu.Unlock()
}

func makeAdvancedHandler(constructor func() (Job, error)) JobHandler {
	return func(jr JobRecord) (any, error) {
		j, err := constructor()
		if err != nil {
			msg := err.Error()
			return &msg, err
		}
		return j.Run(jr)
	}
}

func makeHandler(constructor func(jr JobRecord) (any, error)) JobHandler {
	return func(jr JobRecord) (any, error) {
		return constructor(jr)
	}
}

// getHandler returns the JobHandler for the given operation or an error if not found.
func (w *Worker) getHandler(op Operation) (JobHandler, error) {
	handler, ok := w.manager.handlers[op]
	if !ok {
		msg := fmt.Sprintf("no handler registered for operation %s", op)
		return nil, fmt.Errorf(msg)
	}
	return handler, nil
}
