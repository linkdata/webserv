package webserv_test

import (
	"bytes"
	"log/slog"
	"os"
	"testing"

	"github.com/linkdata/webserv"
)

func TestConfig_Apply(t *testing.T) {
	withCertFiles(t, func(destdir string) {
		homeDir := os.Getenv("HOME")
		if st, err := os.Stat(homeDir); err != nil || !st.IsDir() {
			homeDir = ""
		}
		cfg := &webserv.Config{
			CertDir:     destdir,
			User:        os.Getenv("USER"),
			DataDir:     homeDir,
			DataDirMode: 0750,
		}
		var buf bytes.Buffer
		l, err := cfg.Apply(slog.New(slog.NewTextHandler(&buf, nil)))
		if err != nil {
			t.Error(err)
		}
		if l != nil {
			t.Logf("Apply():\n%#+v\n%s", cfg, buf.String())
			l.Close()
		}
	})
}
