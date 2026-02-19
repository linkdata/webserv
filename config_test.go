package webserv_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
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
