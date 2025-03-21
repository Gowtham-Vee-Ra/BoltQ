package logger

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

var (
	infoLogger  = log.New(os.Stdout, "", 0)
	errorLogger = log.New(os.Stderr, "", 0)
)

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp string      `json:"timestamp"`
	Level     string      `json:"level"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
}

// Info logs an informational message
func Info(msg *string) {
	logMessage("INFO", *msg, nil)
}

// InfoWithData logs an informational message with additional data
func InfoWithData(msg *string, data interface{}) {
	logMessage("INFO", *msg, data)
}

// Error logs an error message
func Error(msg *string) {
	logMessage("ERROR", *msg, nil)
}

// ErrorWithData logs an error message with additional data
func ErrorWithData(msg *string, data interface{}) {
	logMessage("ERROR", *msg, data)
}

// Fatal logs a fatal error message and exits the program
func Fatal(msg *string) {
	logMessage("FATAL", *msg, nil)
	os.Exit(1)
}

// logMessage creates a structured log entry and logs it
func logMessage(level, msg string, data interface{}) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Message:   msg,
		Data:      data,
	}

	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		// Fallback to simple logging if JSON marshaling fails
		if level == "ERROR" || level == "FATAL" {
			errorLogger.Printf("[%s] %s", level, msg)
		} else {
			infoLogger.Printf("[%s] %s", level, msg)
		}
		return
	}

	if level == "ERROR" || level == "FATAL" {
		errorLogger.Println(string(jsonBytes))
	} else {
		infoLogger.Println(string(jsonBytes))
	}
}
