package goli

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/germtb/goli/signals"
)

// LogLevel represents the severity of a log message
type LogLevel string

const (
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
)

// LogMessage represents a captured log message
type LogMessage struct {
	Timestamp time.Time
	Level     LogLevel
	Message   string
}

// LogCapture captures log output for display in the TUI
type LogCapture struct {
	messages    signals.Accessor[[]LogMessage]
	setMessages signals.Setter[[]LogMessage]
	maxMessages int
	mu          sync.Mutex

	// Original stdout/stderr for restoration
	origStdout *os.File
	origStderr *os.File

	// Pipes for capturing
	stdoutReader *os.File
	stdoutWriter *os.File
	stderrReader *os.File
	stderrWriter *os.File

	// Stop channel
	stopCh chan struct{}
}

// NewLogCapture creates a new log capture with the specified max message count
func NewLogCapture(maxMessages int) *LogCapture {
	if maxMessages <= 0 {
		maxMessages = 1000
	}

	messages, setMessages := signals.CreateSignal([]LogMessage{})

	return &LogCapture{
		messages:    messages,
		setMessages: setMessages,
		maxMessages: maxMessages,
	}
}

// Start begins capturing stdout and stderr
func (lc *LogCapture) Start() error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	// Save original stdout/stderr
	lc.origStdout = os.Stdout
	lc.origStderr = os.Stderr

	// Create pipes
	var err error
	lc.stdoutReader, lc.stdoutWriter, err = os.Pipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	lc.stderrReader, lc.stderrWriter, err = os.Pipe()
	if err != nil {
		lc.stdoutReader.Close()
		lc.stdoutWriter.Close()
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Redirect stdout/stderr to our pipes
	os.Stdout = lc.stdoutWriter
	os.Stderr = lc.stderrWriter

	// Start reading from pipes
	lc.stopCh = make(chan struct{})

	go lc.readPipe(lc.stdoutReader, LogLevelInfo)
	go lc.readPipe(lc.stderrReader, LogLevelError)

	return nil
}

// readPipe reads from a pipe and adds messages to the capture
func (lc *LogCapture) readPipe(reader *os.File, level LogLevel) {
	buf := make([]byte, 4096)
	for {
		select {
		case <-lc.stopCh:
			return
		default:
			n, err := reader.Read(buf)
			if err != nil {
				if err != io.EOF {
					// Log read error to original stderr if available
					if lc.origStderr != nil {
						fmt.Fprintf(lc.origStderr, "LogCapture read error: %v\n", err)
					}
				}
				return
			}
			if n > 0 {
				lc.addMessage(level, string(buf[:n]))
			}
		}
	}
}

// Stop stops capturing and restores original stdout/stderr
func (lc *LogCapture) Stop() {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	if lc.stopCh != nil {
		close(lc.stopCh)
		lc.stopCh = nil
	}

	// Restore original stdout/stderr
	if lc.origStdout != nil {
		os.Stdout = lc.origStdout
		lc.origStdout = nil
	}
	if lc.origStderr != nil {
		os.Stderr = lc.origStderr
		lc.origStderr = nil
	}

	// Close pipes
	if lc.stdoutWriter != nil {
		lc.stdoutWriter.Close()
		lc.stdoutWriter = nil
	}
	if lc.stdoutReader != nil {
		lc.stdoutReader.Close()
		lc.stdoutReader = nil
	}
	if lc.stderrWriter != nil {
		lc.stderrWriter.Close()
		lc.stderrWriter = nil
	}
	if lc.stderrReader != nil {
		lc.stderrReader.Close()
		lc.stderrReader = nil
	}
}

// addMessage adds a message to the capture
func (lc *LogCapture) addMessage(level LogLevel, message string) {
	msg := LogMessage{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
	}

	signals.SetWith(lc.setMessages, func(prev []LogMessage) []LogMessage {
		next := append(prev, msg)
		if len(next) > lc.maxMessages {
			// Trim to max
			next = next[len(next)-lc.maxMessages:]
		}
		return next
	}, lc.messages)
}

// Log logs a message at the specified level
func (lc *LogCapture) Log(level LogLevel, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	lc.addMessage(level, message)
}

// Debug logs a debug message
func (lc *LogCapture) Debug(format string, args ...any) {
	lc.Log(LogLevelDebug, format, args...)
}

// Info logs an info message
func (lc *LogCapture) Info(format string, args ...any) {
	lc.Log(LogLevelInfo, format, args...)
}

// Warn logs a warning message
func (lc *LogCapture) Warn(format string, args ...any) {
	lc.Log(LogLevelWarn, format, args...)
}

// Error logs an error message
func (lc *LogCapture) Error(format string, args ...any) {
	lc.Log(LogLevelError, format, args...)
}

// Messages returns the current messages (reactive)
func (lc *LogCapture) Messages() []LogMessage {
	return lc.messages()
}

// LastMessages returns the last n messages (reactive)
func (lc *LogCapture) LastMessages(n int) []LogMessage {
	msgs := lc.messages()
	if len(msgs) <= n {
		return msgs
	}
	return msgs[len(msgs)-n:]
}

// Clear clears all captured messages
func (lc *LogCapture) Clear() {
	lc.setMessages([]LogMessage{})
}

// FormatMessage formats a log message for display
func FormatMessage(msg LogMessage) string {
	timeStr := msg.Timestamp.Format("15:04:05.000")
	return fmt.Sprintf("[%s] %-5s %s", timeStr, msg.Level, msg.Message)
}

// WriteToOriginal writes directly to the original stdout (bypassing capture)
// This is useful for TUI rendering
func (lc *LogCapture) WriteToOriginal(p []byte) (n int, err error) {
	lc.mu.Lock()
	orig := lc.origStdout
	lc.mu.Unlock()

	if orig != nil {
		return orig.Write(p)
	}
	return os.Stdout.Write(p)
}

// OriginalStdout returns the original stdout file
func (lc *LogCapture) OriginalStdout() *os.File {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	if lc.origStdout != nil {
		return lc.origStdout
	}
	return os.Stdout
}
