package webserv_test

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"

	"github.com/linkdata/webserv"
)

func TestDefaultDataDir_AbsoluteSuffixCannotEscapeUserConfigDir(t *testing.T) {
	base, err := os.UserConfigDir()
	if err != nil {
		t.Skipf("UserConfigDir unavailable: %v", err)
	}

	got, err := webserv.DefaultDataDir("", "/webserv-audit")
	if err != nil {
		t.Fatal(err)
	}

	base = filepath.Clean(base)
	got = filepath.Clean(got)
	rel, err := filepath.Rel(base, got)
	if err != nil {
		t.Fatal(err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		t.Fatalf("DefaultDataDir escaped UserConfigDir: base=%q got=%q", base, got)
	}
}

func TestDefaultDataDir_DotDotSuffixCannotEscapeUserConfigDir(t *testing.T) {
	got, err := webserv.DefaultDataDir("", "../webserv-audit")
	if err == nil {
		t.Fatalf("expected invalid suffix error, got path %q", got)
	}
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
