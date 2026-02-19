package webserv_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/linkdata/webserv"
)

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

func TestDefaultDataDir_EnvSuffixCanEscapeUserConfigDir(t *testing.T) {
	base, err := os.UserConfigDir()
	if err != nil {
		t.Skipf("UserConfigDir unavailable: %v", err)
	}

	t.Setenv("WEBSERV_AUDIT_ESCAPE", ".."+string(os.PathSeparator)+".."+string(os.PathSeparator)+"webserv-audit-escape")

	dataDir, err := webserv.DefaultDataDir("", "$WEBSERV_AUDIT_ESCAPE")
	if err != nil {
		return
	}
	dataDir, err = webserv.UseDataDir(dataDir, 0)
	if err != nil {
		t.Fatal(err)
	}

	base = filepath.Clean(base)
	dataDir = filepath.Clean(dataDir)
	rel, err := filepath.Rel(base, dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		t.Fatalf("resolved data directory escaped UserConfigDir: base=%q dataDir=%q", base, dataDir)
	}
}
