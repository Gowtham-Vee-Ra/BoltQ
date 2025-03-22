// pkg/logger/logger.go
package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Log levels
const (
	DebugLevel = "DEBUG"
	InfoLevel  = "INFO"
	WarnLevel  = "WARN"
	ErrorLevel = "ERROR"
	FatalLevel = "FATAL"
)

var (
	// Default logger
	defaultLogger *Logger
	// Mutex for thread safety
	mu sync.Mutex
)

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp string      `json:"timestamp"`
	Level     string      `json:"level"`
	Message   string      `json:"message"`
	Caller    string      `json:"caller,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

// Logger represents a logger instance
type Logger struct {
	level        string
	out          io.Writer
	enableJSON   bool
	enableCaller bool
}

// Setup initializes the global logger with the specified log level
func Setup(level string) {
	mu.Lock()
	defer mu.Unlock()

	if level == "" {
		level = InfoLevel
	}

	// Normalize log level to uppercase
	level = strings.ToUpper(level)

	// Validate log level
	switch level {
	case DebugLevel, InfoLevel, WarnLevel, ErrorLevel, FatalLevel:
		// Valid log level
	default:
		level = InfoLevel
	}

	// Create default logger
	defaultLogger = &Logger{
		level:        level,
		out:          os.Stdout,
		enableJSON:   true,
		enableCaller: true,
	}

	// Log setup completion
	defaultLogger.log(InfoLevel, "Logger initialized", nil)
}

// SetOutput sets the output destination for the default logger
func SetOutput(w io.Writer) {
	mu.Lock()
	defer mu.Unlock()

	if defaultLogger == nil {
		// Initialize default logger if not done yet
		Setup(InfoLevel)
	}

	defaultLogger.out = w
}

// EnableJSON turns on/off JSON formatted logging
func EnableJSON(enable bool) {
	mu.Lock()
	defer mu.Unlock()

	if defaultLogger == nil {
		// Initialize default logger if not done yet
		Setup(InfoLevel)
	}

	defaultLogger.enableJSON = enable
}

// EnableCaller turns on/off caller information in logs
func EnableCaller(enable bool) {
	mu.Lock()
	defer mu.Unlock()

	if defaultLogger == nil {
		// Initialize default logger if not done yet
		Setup(InfoLevel)
	}

	defaultLogger.enableCaller = enable
}

// log logs a message at the specified level
func (l *Logger) log(level, msg string, data interface{}) {
	// Check if logging at this level is enabled
	if !l.shouldLog(level) {
		return
	}

	// Get caller information if enabled
	var caller string
	if l.enableCaller {
		_, file, line, ok := runtime.Caller(2)
		if ok {
			// Extract just the file name without the full path
			parts := strings.Split(file, "/")
			file = parts[len(parts)-1]
			caller = fmt.Sprintf("%s:%d", file, line)
		}
	}

	// Create log entry
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Message:   msg,
		Caller:    caller,
		Data:      data,
	}

	// Output log entry
	if l.enableJSON {
		// JSON format
		jsonEntry, err := json.Marshal(entry)
		if err != nil {
			log.Printf("[ERROR] Failed to marshal log entry: %v", err)
			return
		}
		fmt.Fprintln(l.out, string(jsonEntry))
	} else {
		// Plain text format
		callerInfo := ""
		if caller != "" {
			callerInfo = fmt.Sprintf(" [%s]", caller)
		}

		dataStr := ""
		if data != nil {
			dataBytes, err := json.Marshal(data)
			if err == nil {
				dataStr = " " + string(dataBytes)
			}
		}

		fmt.Fprintf(l.out, "[%s] [%s]%s %s%s\n",
			entry.Timestamp,
			entry.Level,
			callerInfo,
			entry.Message,
			dataStr,
		)
	}
}

// shouldLog determines if a message at the given level should be logged
func (l *Logger) shouldLog(level string) bool {
	levelMap := map[string]int{
		DebugLevel: 0,
		InfoLevel:  1,
		WarnLevel:  2,
		ErrorLevel: 3,
		FatalLevel: 4,
	}

	currentLevelValue, ok := levelMap[l.level]
	if !ok {
		return false
	}

	logLevelValue, ok := levelMap[level]
	if !ok {
		return false
	}

	return logLevelValue >= currentLevelValue
}

// Debug logs a debug message
func Debug(msg string) {
	mu.Lock()
	defer mu.Unlock()

	if defaultLogger == nil {
		// Initialize default logger if not done yet
		Setup(InfoLevel)
	}

	defaultLogger.log(DebugLevel, msg, nil)
}

// Info logs an info message
func Info(msg string) {
	mu.Lock()
	defer mu.Unlock()

	if defaultLogger == nil {
		// Initialize default logger if not done yet
		Setup(InfoLevel)
	}

	defaultLogger.log(InfoLevel, msg, nil)
}

// Warn logs a warning message
func Warn(msg string) {
	mu.Lock()
	defer mu.Unlock()

	if defaultLogger == nil {
		// Initialize default logger if not done yet
		Setup(InfoLevel)
	}

	defaultLogger.log(WarnLevel, msg, nil)
}

// Error logs an error message
func Error(msg string) {
	mu.Lock()
	defer mu.Unlock()

	if defaultLogger == nil {
		// Initialize default logger if not done yet
		Setup(InfoLevel)
	}

	defaultLogger.log(ErrorLevel, msg, nil)
}

// Fatal logs a fatal message and exits the program
func Fatal(msg string) {
	mu.Lock()
	defer mu.Unlock()

	if defaultLogger == nil {
		// Initialize default logger if not done yet
		Setup(InfoLevel)
	}

	defaultLogger.log(FatalLevel, msg, nil)
	os.Exit(1)
}

// DebugWithData logs a debug message with additional data
func DebugWithData(msg string, data interface{}) {
	mu.Lock()
	defer mu.Unlock()

	if defaultLogger == nil {
		// Initialize default logger if not done yet
		Setup(InfoLevel)
	}

	defaultLogger.log(DebugLevel, msg, data)
}

// InfoWithData logs an info message with additional data
func InfoWithData(msg string, data interface{}) {
	mu.Lock()
	defer mu.Unlock()

	if defaultLogger == nil {
		// Initialize default logger if not done yet
		Setup(InfoLevel)
	}

	defaultLogger.log(InfoLevel, msg, data)
}

// WarnWithData logs a warning message with additional data
func WarnWithData(msg string, data interface{}) {
	mu.Lock()
	defer mu.Unlock()

	if defaultLogger == nil {
		// Initialize default logger if not done yet
		Setup(InfoLevel)
	}

	defaultLogger.log(WarnLevel, msg, data)
}

// ErrorWithData logs an error message with additional data
func ErrorWithData(msg string, data interface{}) {
	mu.Lock()
	defer mu.Unlock()

	if defaultLogger == nil {
		// Initialize default logger if not done yet
		Setup(InfoLevel)
	}

	defaultLogger.log(ErrorLevel, msg, data)
}

// FatalWithData logs a fatal message with additional data and exits the program
func FatalWithData(msg string, data interface{}) {
	mu.Lock()
	defer mu.Unlock()

	if defaultLogger == nil {
		// Initialize default logger if not done yet
		Setup(InfoLevel)
	}

	defaultLogger.log(FatalLevel, msg, data)
	os.Exit(1)
}
