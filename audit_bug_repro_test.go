package webserv_test

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/linkdata/webserv"
)

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
