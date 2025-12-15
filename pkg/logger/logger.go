package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"pdf-text-reader/internal/domain"
)

// LogLevel represents different logging levels
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// AppLogger implements the domain.Logger interface
type AppLogger struct {
	level  LogLevel
	logger *log.Logger
}

// NewLogger creates a new logger instance
func NewLogger(levelStr string) domain.Logger {
	level := parseLogLevel(levelStr)
	logger := log.New(os.Stdout, "", 0)
	
	return &AppLogger{
		level:  level,
		logger: logger,
	}
}

// Info logs an info message
func (l *AppLogger) Info(msg string, fields ...interface{}) {
	if l.level <= INFO {
		l.log("INFO", msg, fields...)
	}
}

// Error logs an error message
func (l *AppLogger) Error(msg string, err error, fields ...interface{}) {
	if l.level <= ERROR {
		allFields := append([]interface{}{"error", err}, fields...)
		l.log("ERROR", msg, allFields...)
	}
}

// Debug logs a debug message
func (l *AppLogger) Debug(msg string, fields ...interface{}) {
	if l.level <= DEBUG {
		l.log("DEBUG", msg, fields...)
	}
}

// Warn logs a warning message
func (l *AppLogger) Warn(msg string, fields ...interface{}) {
	if l.level <= WARN {
		l.log("WARN", msg, fields...)
	}
}

// log is the internal logging method
func (l *AppLogger) log(level, msg string, fields ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	
	logMsg := fmt.Sprintf("[%s] %s: %s", timestamp, level, msg)
	
	if len(fields) > 0 {
		fieldStrs := make([]string, 0, len(fields)/2)
		for i := 0; i < len(fields); i += 2 {
			if i+1 < len(fields) {
				fieldStrs = append(fieldStrs, fmt.Sprintf("%v=%v", fields[i], fields[i+1]))
			}
		}
		if len(fieldStrs) > 0 {
			logMsg += " " + strings.Join(fieldStrs, " ")
		}
	}
	
	l.logger.Println(logMsg)
}

// parseLogLevel converts string log level to LogLevel enum
func parseLogLevel(levelStr string) LogLevel {
	switch strings.ToLower(levelStr) {
	case "debug":
		return DEBUG
	case "info":
		return INFO
	case "warn", "warning":
		return WARN
	case "error":
		return ERROR
	default:
		return INFO
	}
}
