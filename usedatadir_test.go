package webserv_test

import (
	"os"
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
	want, err := filepath.Abs("foo")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("DefaultDataDir(%q) = %q, want %q", "foo", got, want)
	}

	got, err = webserv.DefaultDataDir("", "")
	if err != nil {
		t.Error(err)
	}
	if got != "" {
		t.Errorf("want empty data dir, got %q", got)
	}
}

func TestDefaultDataDir_ExpandsEnv(t *testing.T) {
	// DefaultDataDir expands environment variables before absolutizing, so a
	// "$VAR" in dataDir resolves to the variable's value.
	dir := t.TempDir()
	t.Setenv("WEBSERV_TEST_DATADIR", dir)

	got, err := webserv.DefaultDataDir("$WEBSERV_TEST_DATADIR/sub", "")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "sub"); got != want {
		t.Fatalf("DefaultDataDir expanded to %q, want %q", got, want)
	}
}

func TestDefaultDataDir_ExpandToEmptyIsNotCwd(t *testing.T) {
	// A non-empty dataDir that expands to empty (an unset variable) must yield
	// an empty result, never the current working directory via filepath.Abs("").
	t.Setenv("WEBSERV_TEST_UNSET", "")

	for _, suffix := range []string{"", "suffix"} {
		got, err := webserv.DefaultDataDir("$WEBSERV_TEST_UNSET", suffix)
		if err != nil {
			t.Fatalf("DefaultDataDir(%q, %q) error: %v", "$WEBSERV_TEST_UNSET", suffix, err)
		}
		if got != "" {
			t.Fatalf("DefaultDataDir(%q, %q) = %q, want empty string", "$WEBSERV_TEST_UNSET", suffix, got)
		}
	}
}

func TestDefaultDataDir_DoesNotExpandUserConfigDir(t *testing.T) {
	// The os.UserConfigDir base is system-provided: a literal "$" in it (a valid
	// path character) must not be passed through os.ExpandEnv. Only the
	// caller-supplied suffix is expanded.
	configHome := filepath.Join(t.TempDir(), "cfg$svc")
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("svc", "WRONGLY_EXPANDED")

	base, err := os.UserConfigDir()
	if err != nil {
		t.Skipf("UserConfigDir unavailable: %v", err)
	}
	if base != configHome {
		t.Skipf("UserConfigDir does not honor XDG_CONFIG_HOME on this platform: %q", base)
	}

	got, err := webserv.DefaultDataDir("", "myapp")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(base, "myapp"); got != want {
		t.Fatalf("DefaultDataDir mangled UserConfigDir:\n got  = %q\n want = %q", got, want)
	}
}

func TestUseDataDir(t *testing.T) {
	got, err := webserv.UseDataDir(".", 0o750)
	if err != nil {
		t.Error(err)
	}
	if got == "" || got == "." {
		t.Error("failed to expand path")
	}
}

func TestUseDataDir_DoesNotDoubleExpandEnv(t *testing.T) {
	// UseDataDir does not expand environment variables, so a literal "$" in the
	// path survives unchanged.
	dir := t.TempDir()
	literalDollar := filepath.Join(dir, "$NOTAVAR")
	if err := os.MkdirAll(literalDollar, 0o750); err != nil {
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

	got, err := webserv.DefaultDataDir("", "/webserv-config")
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
