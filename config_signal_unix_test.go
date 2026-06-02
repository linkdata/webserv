//go:build unix

package webserv_test

import (
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/linkdata/webserv"
)

func TestConfigServeWith_SecondSignalUsesDefaultBehavior(t *testing.T) {
	if os.Getenv("WEBSERV_SECOND_SIGNAL_CHILD") == "1" {
		runSecondSignalChild(t)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestConfigServeWith_SecondSignalUsesDefaultBehavior$")
	cmd.Env = append(os.Environ(), "WEBSERV_SECOND_SIGNAL_CHILD=1")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("child survived second SIGTERM; output:\n%s", output)
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("child run error = %v (%T), want ExitError; output:\n%s", err, err, output)
	}
	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok {
		t.Fatalf("child status = %T, want syscall.WaitStatus; output:\n%s", exitErr.Sys(), output)
	}
	if !status.Signaled() || status.Signal() != syscall.SIGTERM {
		t.Fatalf("child exit status = %v, want SIGTERM; output:\n%s", status, output)
	}
}

func runSecondSignalChild(t *testing.T) {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()

	started := make(chan struct{})
	unblock := make(chan struct{})
	var startedOnce sync.Once
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			startedOnce.Do(func() {
				close(started)
			})
			<-unblock
			_, _ = w.Write([]byte("ok"))
		}),
	}

	logger := newNotifyingLogger(io.Discard, "webserv: stopped")
	cfg := &webserv.Config{
		Logger: logger,
	}
	done := make(chan error, 1)
	go func() {
		done <- cfg.ServeWith(t.Context(), srv, l)
	}()

	reqDone := make(chan struct{})
	go func() {
		defer close(reqDone)
		if resp, err := http.Get("http://" + l.Addr().String()); err == nil {
			defer func() { _ = resp.Body.Close() }()
			_, _ = io.Copy(io.Discard, resp.Body)
		}
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for handler to start")
	}

	if err = signalSelf(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case <-logger.ready:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first signal to start shutdown")
	}
	if err = signalSelf(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}

	select {
	case <-done:
		close(unblock)
		<-reqDone
		os.Exit(0)
	case <-time.After(1500 * time.Millisecond):
		close(unblock)
		<-reqDone
		os.Exit(0)
	}
}
