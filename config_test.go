package webserv_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/linkdata/webserv"
)

func TestConfig_ListenAndServe_Signalled(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process signalling from tests is not reliable on windows")
	}
	withCertFiles(t, func(destdir string) {
		homeDir := os.Getenv("HOME")
		if st, err := os.Stat(homeDir); err != nil || !st.IsDir() {
			homeDir = ""
		}
		var buf bytes.Buffer
		cfg := &webserv.Config{
			CertDir:     destdir,
			User:        os.Getenv("USER"),
			DataDir:     homeDir,
			DataDirMode: 0750,
			Logger:      slog.New(slog.NewTextHandler(&buf, nil)),
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		go func() {
			for {
				time.Sleep(50 * time.Millisecond)
				if p, err := os.FindProcess(os.Getpid()); err == nil {
					if err = p.Signal(syscall.SIGTERM); err != nil {
						t.Error(err)
					}
					return
				}
			}
		}()
		err := cfg.ListenAndServe(ctx, nil)
		if err != nil {
			t.Error(err)
		}
		s := buf.String()
		t.Log(s)
		if !strings.Contains(s, "signal") {
			t.Error("expected 'signal' in log output")
		}
		if !strings.Contains(s, "terminated") {
			t.Error("expected 'terminated' signal name in log output")
		}
	})
}

func TestConfig_ListenAndServe_Cancelled(t *testing.T) {
	withCertFiles(t, func(destdir string) {
		homeDir := os.Getenv("HOME")
		if st, err := os.Stat(homeDir); err != nil || !st.IsDir() {
			homeDir = ""
		}
		var buf bytes.Buffer
		cfg := &webserv.Config{
			CertDir:     destdir,
			User:        os.Getenv("USER"),
			DataDir:     homeDir,
			DataDirMode: 0750,
			Logger:      slog.New(slog.NewTextHandler(&buf, nil)),
		}
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()
		err := cfg.ListenAndServe(ctx, nil)
		if !errors.Is(err, context.Canceled) {
			t.Error(err)
		}
		s := buf.String()
		t.Log(s)
		if !strings.Contains(s, "context canceled") {
			t.Error("expected 'context canceled' in log output")
		}
	})
}

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

func TestConfigListen_ErrorClearsListenURL(t *testing.T) {
	const initialListenURL = "https://example.invalid:443"
	cfg := &webserv.Config{
		Address:   "127.0.0.1:99999",
		ListenURL: initialListenURL,
	}

	l, err := cfg.Listen()
	if l != nil {
		_ = l.Close()
	}
	if err == nil {
		t.Fatal("expected Listen() error")
	}
	if cfg.ListenURL != "" {
		t.Fatalf("ListenURL not cleared on error: got %q", cfg.ListenURL)
	}
}

func TestConfigListen_ErrorStillAbsolutizesCertDir(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	certDir := t.TempDir()
	relCertDir, err := filepath.Rel(cwd, certDir)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &webserv.Config{
		CertDir: relCertDir,
	}

	l, err := cfg.Listen()
	if l != nil {
		_ = l.Close()
	}
	if err == nil {
		t.Fatal("expected Listen() error for missing cert files")
	}
	absCertDir, err := filepath.Abs(relCertDir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CertDir != absCertDir {
		t.Fatalf("CertDir not absolutized on error: got %q want %q", cfg.CertDir, absCertDir)
	}
}

func TestConfigServeWith_NilListenerPanics(t *testing.T) {
	cfg := &webserv.Config{}
	srv := &http.Server{}

	panicked := false
	defer func() {
		if r := recover(); r != nil {
			panicked = true
		}
		if !panicked {
			t.Fatal("expected ServeWith() to panic for nil listener")
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_ = cfg.ServeWith(ctx, srv, nil)
}

func TestConfigListen_ErrorAfterBindMayPopulateListenURL(t *testing.T) {
	const noSuchUser = "webserv-no-such-user-audit-listenurl"
	if _, err := user.Lookup(noSuchUser); err == nil {
		t.Skipf("test user unexpectedly exists: %q", noSuchUser)
	}

	cfg := &webserv.Config{
		Address: "127.0.0.1:0",
		User:    noSuchUser,
	}

	l, err := cfg.Listen()
	if l != nil {
		_ = l.Close()
	}
	if err == nil {
		t.Fatal("expected Listen() error")
	}
	if cfg.ListenURL == "" {
		t.Fatal("expected ListenURL to be populated after bind, even though Listen() failed later")
	}
}

func TestConfigListen_EmptyDataDirStaysEmpty(t *testing.T) {
	cfg := &webserv.Config{
		Address: "127.0.0.1:0",
	}

	l, err := cfg.Listen()
	if l != nil {
		_ = l.Close()
	}
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DataDir != "" {
		t.Fatalf("expected empty DataDir, got %q", cfg.DataDir)
	}
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

func TestConfigServeWith_SignalShutdownCanHangWithoutDeadline(t *testing.T) {
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
	defer func() { _ = srv.Close() }()

	savedTimeLimit := webserv.ShutdownTimeLimit
	defer func() { webserv.ShutdownTimeLimit = savedTimeLimit }()
	webserv.ShutdownTimeLimit = time.Millisecond * 10

	done := make(chan error, 1)
	go func() {
		done <- (&webserv.Config{}).ServeWith(context.Background(), srv, l)
	}()

	go func() {
		defer close(reqDone)
		client := &http.Client{Timeout: 2 * webserv.ShutdownTimeLimit}
		_, _ = client.Get("http://" + l.Addr().String())
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for handler to start")
	}

	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if err = p.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}

	select {
	case err = <-done:
	case <-time.After(1500 * time.Millisecond):
		close(unblock)
		<-reqDone
		t.Fatal("ServeWith() did not return within 1.5s after SIGTERM")
	}

	close(unblock)
	<-reqDone

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("ServeWith() error = %v, want %v", err, context.DeadlineExceeded)
	}
}
