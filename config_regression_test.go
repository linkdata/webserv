package webserv_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os/user"
	"testing"
	"time"

	"github.com/linkdata/webserv"
)

func TestConfigListen_ErrorDoesNotReturnListener(t *testing.T) {
	// Pick a port that should be free for this test run.
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := probe.Addr().String()
	if err := probe.Close(); err != nil {
		t.Fatal(err)
	}

	const noSuchUser = "webserv-no-such-user-regression-test"
	if _, err := user.Lookup(noSuchUser); err == nil {
		t.Skipf("test user unexpectedly exists: %q", noSuchUser)
	}

	cfg := &webserv.Config{
		Address: addr,
		User:    noSuchUser,
	}
	l, err := cfg.Listen()
	if err == nil {
		if l != nil {
			_ = l.Close()
		}
		t.Fatal("expected Listen error")
	}
	if l != nil {
		defer func() { _ = l.Close() }()
		t.Fatalf("expected nil listener on Listen error, got %s", l.Addr().String())
	}
}

func TestConfigServeWith_ExternalCloseReturnsPromptly(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()

	cfg := &webserv.Config{}
	srv := &http.Server{}
	ctx, cancel := context.WithTimeout(context.Background(), 750*time.Millisecond)
	defer cancel()

	go func() {
		deadline := time.Now().Add(250 * time.Millisecond)
		for {
			conn, dialErr := net.DialTimeout("tcp", l.Addr().String(), 20*time.Millisecond)
			if dialErr == nil {
				_ = conn.Close()
				break
			}
			if time.Now().After(deadline) {
				return
			}
			time.Sleep(time.Millisecond)
		}
		_ = srv.Close()
	}()

	start := time.Now()
	err = cfg.ServeWith(ctx, srv, l)
	elapsed := time.Since(start)
	if errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("ServeWith() blocked until context timeout (%v)", elapsed)
	}
	if elapsed > 350*time.Millisecond {
		t.Fatalf("ServeWith() took too long to return after external close: %v", elapsed)
	}
}
