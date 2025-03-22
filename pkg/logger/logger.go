// pkg/logger/logger.go
package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Level string

const (
	InfoLevel  Level = "INFO"
	ErrorLevel Level = "ERROR"
	WarnLevel  Level = "WARN"
	DebugLevel Level = "DEBUG"
)

type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Component string                 `json:"component"`
	JobID     string                 `json:"job_id,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// Logger represents a structured logger
type Logger struct {
	component string
}

// NewLogger creates a new logger for a specific component
func NewLogger(component string) *Logger {
	return &Logger{
		component: component,
	}
}

// log writes a log entry to stdout
func (l *Logger) log(level Level, msg string, jobID string, data map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     string(level),
		Message:   msg,
		Component: l.component,
		JobID:     jobID,
		Data:      data,
	}

	jsonData, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling log entry: %v\n", err)
		return
	}

	fmt.Fprintln(os.Stdout, string(jsonData))
}

// Info logs an info message
func (l *Logger) Info(msg string, data ...map[string]interface{}) {
	var extras map[string]interface{}
	if len(data) > 0 {
		extras = data[0]
	}
	l.log(InfoLevel, msg, "", extras)
}

// Error logs an error message
func (l *Logger) Error(msg string, data ...map[string]interface{}) {
	var extras map[string]interface{}
	if len(data) > 0 {
		extras = data[0]
	}
	l.log(ErrorLevel, msg, "", extras)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, data ...map[string]interface{}) {
	var extras map[string]interface{}
	if len(data) > 0 {
		extras = data[0]
	}
	l.log(WarnLevel, msg, "", extras)
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, data ...map[string]interface{}) {
	var extras map[string]interface{}
	if len(data) > 0 {
		extras = data[0]
	}
	l.log(DebugLevel, msg, "", extras)
}

// WithJob returns a log message with job context
func (l *Logger) WithJob(jobID string, msg string, data ...map[string]interface{}) {
	var extras map[string]interface{}
	if len(data) > 0 {
		extras = data[0]
	}
	l.log(InfoLevel, msg, jobID, extras)
}

// JobError logs an error message with job context
func (l *Logger) JobError(jobID string, msg string, data ...map[string]interface{}) {
	var extras map[string]interface{}
	if len(data) > 0 {
		extras = data[0]
	}
	l.log(ErrorLevel, msg, jobID, extras)
}
