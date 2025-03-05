package taskflow

// Job is the interface each job must implement.
type Job interface {
	Run() error
	GetOutput() *string
}
