package webserv_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/linkdata/webserv"
)

func TestLoadCert_RejectsPathTraversalInFileOverrides(t *testing.T) {
	withCertFiles(t, func(certRoot string) {
		sandbox := filepath.Join(certRoot, "sandbox")
		if err := os.MkdirAll(sandbox, 0755); err != nil {
			t.Fatal(err)
		}

		// The override arguments are documented as file names.
		// ".." should not be able to escape certDir.
		cert, absDir, err := webserv.LoadCert(sandbox, "../fullchain.pem", "../privkey.pem")
		if err == nil {
			t.Fatalf("expected path traversal error, got cert=%v absDir=%q", cert != nil, absDir)
		}
		if cert != nil {
			t.Fatalf("expected nil cert on traversal attempt, got non-nil cert from %q", absDir)
		}
	})
}

func TestConfigServeWith_PropagatesShutdownContextError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process signalling from tests is not reliable on windows")
	}

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()

	started := make(chan struct{})
	unblock := make(chan struct{})
	reqDone := make(chan struct{})

	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			close(started)
			<-unblock
			_, _ = w.Write([]byte("ok"))
		}),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	go func() {
		defer close(reqDone)
		_, _ = http.Get("http://" + l.Addr().String())
	}()

	go func() {
		<-started
		if p, findErr := os.FindProcess(os.Getpid()); findErr == nil {
			_ = p.Signal(syscall.SIGTERM)
		}
	}()

	err = (&webserv.Config{}).ServeWith(ctx, srv, l)
	close(unblock)
	<-reqDone

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("ServeWith() error = %v, want %v", err, context.DeadlineExceeded)
	}
}
