package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type Logger struct {
	mu        sync.Mutex
	writer    io.Writer
	debugMode bool
}

func New(w io.Writer, debug bool) *Logger {
	return &Logger{writer: w, debugMode: debug}
}

type LogEntry struct {
	Level     string    `json:"level"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	Data      any       `json:"data,omitempty"`
}

func (l *Logger) Info(msg string, data any) {
	l.log("INFO", msg, data)
}

func (l *Logger) Error(msg string, data any) {
	l.log("ERROR", msg, data)
}

func (l *Logger) Debug(msg string, data any) {
	if l.debugMode {
		l.log("DEBUG", msg, data)
	}
}

func (l *Logger) log(level, msg string, data any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := LogEntry{
		Level:     level,
		Timestamp: time.Now(),
		Message:   msg,
		Data:      data,
	}

	bytes, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Logger error: %v\n", err)
		return
	}
	fmt.Fprintln(l.writer, string(bytes))
}