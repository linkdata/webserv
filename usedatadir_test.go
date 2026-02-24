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
	want, _ := filepath.Abs(path.Join("foo"))
	if got != want {
		t.Error(got)
	}

	got, err = webserv.DefaultDataDir("", "")
	if err != nil {
		t.Error(err)
	}
	if got != "" {
		t.Errorf("want empty data dir, got %q", got)
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

func TestUseDataDir_DoesNotDoubleExpandEnv(t *testing.T) {
	// Bug: DefaultDataDir already calls os.ExpandEnv, then UseDataDir
	// calls os.ExpandEnv again. If the path contains a literal '$' after
	// the first expansion, the second expansion misinterprets it.
	dir := t.TempDir()
	// UseDataDir should not expand an already-absolute path further.
	// A literal "$" in the path should survive unchanged.
	literalDollar := filepath.Join(dir, "$NOTAVAR")
	if err := os.MkdirAll(literalDollar, 0750); err != nil {
		t.Fatal(err)
	}
	got, err := webserv.UseDataDir(literalDollar, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got != literalDollar {
		t.Fatalf("UseDataDir expanded literal $: got %q, want %q", got, literalDollar)
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
