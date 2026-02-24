package webserv

import (
	"testing"
)

type recordingLogger struct {
	messages []string
}

func (l *recordingLogger) Info(msg string, keyValuePairs ...any)  { l.messages = append(l.messages, msg) }
func (l *recordingLogger) Warn(msg string, keyValuePairs ...any)  {}
func (l *recordingLogger) Error(msg string, keyValuePairs ...any) {}

func TestLogInfo_WithoutKeyValuePairsIsNotSuppressed(t *testing.T) {
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

func TestLogInfo_EmptyStringValueIsSuppressed(t *testing.T) {
	rl := &recordingLogger{}
	cfg := &Config{Logger: rl}
	cfg.logInfo("loaded certificates", "dir", "")
	if len(rl.messages) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(rl.messages))
	}
}

func TestLogInfo_NonEmptyStringValueIsNotSuppressed(t *testing.T) {
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
