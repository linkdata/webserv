package webserv_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/linkdata/webserv"
)

func TestConfig_ListenAndServe_Signalled(t *testing.T) {
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
			waited := 0
			for cfg.BreakChan() == nil {
				if waited > 500 {
					t.Error("timeout waiting for server to start")
					return
				}
			}
			cfg.BreakChan() <- syscall.SIGUSR1
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
			defer cancel()
			waited := 0
			for cfg.BreakChan() == nil {
				if waited > 500 {
					t.Error("timeout waiting for server to start")
					return
				}
			}
		}()
		err := cfg.ListenAndServe(ctx, nil)
		if !errors.Is(err, context.Canceled) {
			t.Error(err)
		}
		s := buf.String()
		t.Log(s)
		if !strings.Contains(s, "context done") {
			t.Error("expected 'context done' in log output")
		}
	})
}
