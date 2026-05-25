package webserv

import (
	"bytes"
	"log"
	"testing"
)

func TestTLSErrorLogWriter_DropsTLSHandshakeErrors(t *testing.T) {
	var buf bytes.Buffer
	w := tlsErrorLogWriter{
		previous: log.New(&buf, "", 0),
	}
	msg := []byte(tlsHandshakeErrorLogPrefix + "127.0.0.1:1234: EOF\n")

	n, err := w.Write(msg)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(msg) {
		t.Fatalf("Write() n = %d, want %d", n, len(msg))
	}
	if got := buf.String(); got != "" {
		t.Fatalf("Write() forwarded TLS handshake error: %q", got)
	}
}

func TestTLSErrorLogWriter_ForwardsOtherErrors(t *testing.T) {
	var buf bytes.Buffer
	w := tlsErrorLogWriter{
		previous: log.New(&buf, "", 0),
	}
	msg := []byte("http: Accept error: boom\n")

	n, err := w.Write(msg)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(msg) {
		t.Fatalf("Write() n = %d, want %d", n, len(msg))
	}
	if got, want := buf.String(), string(msg); got != want {
		t.Fatalf("Write() forwarded %q, want %q", got, want)
	}
}

func TestTLSErrorLogWriter_ForwardsToStandardLogger(t *testing.T) {
	var buf bytes.Buffer
	oldFlags := log.Flags()
	oldOutput := log.Writer()
	oldPrefix := log.Prefix()
	log.SetFlags(0)
	log.SetOutput(&buf)
	log.SetPrefix("standard: ")
	t.Cleanup(func() {
		log.SetFlags(oldFlags)
		log.SetOutput(oldOutput)
		log.SetPrefix(oldPrefix)
	})

	msg := []byte("http: Accept error: boom\n")
	w := tlsErrorLogWriter{}

	n, err := w.Write(msg)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(msg) {
		t.Fatalf("Write() n = %d, want %d", n, len(msg))
	}
	if got, want := buf.String(), "standard: "+string(msg); got != want {
		t.Fatalf("Write() forwarded %q, want %q", got, want)
	}
}

func TestTLSErrorLogWriter_PreservesPreviousLoggerPrefix(t *testing.T) {
	var buf bytes.Buffer
	w := tlsErrorLogWriter{
		previous: log.New(&buf, "previous: ", 0),
	}
	msg := []byte("http: Accept error: boom\n")

	n, err := w.Write(msg)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(msg) {
		t.Fatalf("Write() n = %d, want %d", n, len(msg))
	}
	if got, want := buf.String(), "previous: "+string(msg); got != want {
		t.Fatalf("Write() forwarded %q, want %q", got, want)
	}
}
