package webserv_test

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/linkdata/webserv"
)

func TestDefaultDataDir(t *testing.T) {
	got, err := webserv.DefaultDataDir("", "suffix")
	if err != nil {
		t.Error(err)
	}
	if !strings.HasSuffix(got, "suffix") {
		t.Error(got)
	}

	got, err = webserv.DefaultDataDir("foo", "suffix")
	if err != nil {
		t.Error(err)
	}
	want, _ := filepath.Abs(path.Join("foo", "suffix"))
	if got != want {
		t.Error(got)
	}
}

func TestUseDataDir(t *testing.T) {
	got, err := webserv.UseDataDir(".", 0750)
	if err != nil {
		t.Error(err)
	}
	if got == "" || got == "." {
		t.Error("failed to expand path")
	}
}

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
