package webserv

import (
	"testing"
	"time"
)

type recordingLogger struct {
	messages []string
}

func (l *recordingLogger) Info(msg string, keyValuePairs ...any) {
	l.messages = append(l.messages, msg)
}
func (l *recordingLogger) Warn(msg string, keyValuePairs ...any)  {}
func (l *recordingLogger) Error(msg string, keyValuePairs ...any) {}

func TestLogInfo_LogsWithoutKeyValuePairs(t *testing.T) {
	rl := &recordingLogger{}
	cfg := &Config{Logger: rl}
	cfg.logInfo("server ready")
	if len(rl.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(rl.messages))
	}
	if rl.messages[0] != "webserv: server ready" {
		t.Fatalf("unexpected message: %q", rl.messages[0])
	}
}

func TestLogInfo_LogsKeyValuePairs(t *testing.T) {
	rl := &recordingLogger{}
	cfg := &Config{Logger: rl}
	cfg.logInfo("loaded certificates", "dir", "/tmp")
	if len(rl.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(rl.messages))
	}
}

func TestLogInfo_NilLoggerDoesNotPanic(t *testing.T) {
	cfg := &Config{}
	cfg.logInfo("should not panic")
	cfg.logInfo("should not panic", "key", "value")
}

func TestShutdownTimeLimit_ZeroUsesDefault(t *testing.T) {
	cfg := &Config{}
	if got := cfg.shutdownTimeLimit(); got != defaultShutdownTimeLimit {
		t.Fatalf("shutdownTimeLimit() = %v, want %v", got, defaultShutdownTimeLimit)
	}
}

func TestShutdownTimeLimit_ConfigValueOverridesDefault(t *testing.T) {
	cfg := &Config{ShutdownTimeLimit: 25 * time.Millisecond}
	if got := cfg.shutdownTimeLimit(); got != cfg.ShutdownTimeLimit {
		t.Fatalf("shutdownTimeLimit() = %v, want %v", got, cfg.ShutdownTimeLimit)
	}
}

// TestListen_DoesNotLogEmptyValues verifies that the optional setup log lines
// are suppressed at the call site when their values (certificate directory,
// user, data directory) are empty.
func TestListen_DoesNotLogEmptyValues(t *testing.T) {
	rl := &recordingLogger{}
	cfg := &Config{Address: "127.0.0.1:0", Logger: rl}
	l, err := cfg.Listen()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()
	if len(rl.messages) != 0 {
		t.Fatalf("expected no setup log lines for empty values, got %v", rl.messages)
	}
}
