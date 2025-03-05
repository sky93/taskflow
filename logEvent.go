package taskflow

import (
	"fmt"
	"os"
)

func defaultInfoLog(ev LogEvent) {
	// Simple fallback to stdout
	msg := fmt.Sprintf("[taskflow:INFO] %s", ev.Message)
	if ev.Err != nil {
		msg += fmt.Sprintf(" | error: %v", ev.Err)
	}
	_, _ = fmt.Fprintln(os.Stdout, msg)
}

func defaultErrorLog(ev LogEvent) {
	// Simple fallback to stderr
	msg := fmt.Sprintf("[taskflow:ERROR] %s", ev.Message)
	if ev.Err != nil {
		msg += fmt.Sprintf(" | error: %v", ev.Err)
	}
	_, _ = fmt.Fprintln(os.Stderr, msg)
}

// Helper methods to invoke logging
func (c *Config) logInfo(ev LogEvent) {
	if c.InfoLog == nil {
		defaultInfoLog(ev)
		return
	}
	c.InfoLog(ev)
}

func (c *Config) logError(ev LogEvent) {
	if c.ErrorLog == nil {
		defaultErrorLog(ev)
		return
	}
	c.ErrorLog(ev)
}
