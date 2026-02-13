package common

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

// Logger defines the logging interface used throughout secs4go.
//
// Library users can provide their own implementation or use the built-in helpers:
//   - NopLogger()     — silent, zero overhead (default when no logger is configured)
//   - NewStdLogger()  — wraps Go's standard log package
type Logger interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
}

// ---------------------------------------------------------------------------
// nopLogger – default, zero-allocation, silent logger
// ---------------------------------------------------------------------------

type nopLogger struct{}

func (nopLogger) Debug(string, ...interface{}) {}
func (nopLogger) Info(string, ...interface{})  {}
func (nopLogger) Warn(string, ...interface{})  {}
func (nopLogger) Error(string, ...interface{}) {}

// NopLogger returns a Logger that discards all output.
// This is the default when the user does not supply a Logger.
func NopLogger() Logger { return nopLogger{} }

// ---------------------------------------------------------------------------
// stdLogger – adapts Go's standard log.Logger to the Logger interface
// ---------------------------------------------------------------------------

type stdLogger struct {
	l *log.Logger
}

func (s *stdLogger) Debug(msg string, kv ...interface{}) {
	s.l.Println("[DEBUG]", formatLogMsg(msg, kv))
}
func (s *stdLogger) Info(msg string, kv ...interface{}) {
	s.l.Println("[INFO]", formatLogMsg(msg, kv))
}
func (s *stdLogger) Warn(msg string, kv ...interface{}) {
	s.l.Println("[WARN]", formatLogMsg(msg, kv))
}
func (s *stdLogger) Error(msg string, kv ...interface{}) {
	s.l.Println("[ERROR]", formatLogMsg(msg, kv))
}

// NewStdLogger creates a Logger backed by Go's standard log package.
// If writer is nil, os.Stderr is used.
func NewStdLogger(writer io.Writer, prefix string) Logger {
	if writer == nil {
		writer = os.Stderr
	}
	return &stdLogger{l: log.New(writer, prefix, log.LstdFlags)}
}

// formatLogMsg builds a human-readable string from a message and key-value pairs.
func formatLogMsg(msg string, kv []interface{}) string {
	if len(kv) == 0 {
		return msg
	}
	var b strings.Builder
	b.WriteString(msg)
	for i := 0; i+1 < len(kv); i += 2 {
		b.WriteString(fmt.Sprintf(" %v=%v", kv[i], kv[i+1]))
	}
	if len(kv)%2 != 0 {
		b.WriteString(fmt.Sprintf(" EXTRA=%v", kv[len(kv)-1]))
	}
	return b.String()
}
