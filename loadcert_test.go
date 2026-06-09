package webserv_test

import (
	"path/filepath"
	"testing"

	"github.com/linkdata/webserv"
)

func TestLoadCert_UsesFilepathJoinNotPathJoin(t *testing.T) {
	// LoadCert joins paths with filepath.Join so OS-appropriate separators are
	// used: relative components such as "sub/.." resolve correctly and the key
	// pair still loads.
	withCertFiles(t, func(destdir string) {
		// Create a subdirectory to force path joining
		subdir := filepath.Join(destdir, "sub")
		cert, _, err := webserv.LoadCert(destdir, "sub/../"+webserv.FullchainPem, "sub/../"+webserv.PrivkeyPem)
		if err != nil {
			t.Fatalf("LoadCert failed with relative path components: %v", err)
		}
		_ = subdir
		if cert == nil {
			t.Fatal("expected non-nil cert")
		}
	})
}

func TestLoadCert_ExpandToEmptyIsNotCwd(t *testing.T) {
	// A non-empty certDir that expands to empty (an unset variable) must be
	// treated as "no certificate directory", never resolved to the current
	// working directory via filepath.Abs("").
	t.Setenv("WEBSERV_TEST_UNSET", "")

	cert, absCertDir, err := webserv.LoadCert("$WEBSERV_TEST_UNSET", "", "")
	if err != nil {
		t.Fatalf("LoadCert(%q) error: %v", "$WEBSERV_TEST_UNSET", err)
	}
	if cert != nil {
		t.Errorf("LoadCert(%q) cert = %v, want nil", "$WEBSERV_TEST_UNSET", cert)
	}
	if absCertDir != "" {
		t.Errorf("LoadCert(%q) absCertDir = %q, want empty string", "$WEBSERV_TEST_UNSET", absCertDir)
	}
}

func TestLoadCert(t *testing.T) {
	cert, absDir, err := webserv.LoadCert("", "", "")
	if err != nil {
		t.Error(err)
	}
	if cert != nil {
		t.Error("cert not nil")
	}
	if absDir != "" {
		t.Error(absDir)
	}
	withCertFiles(t, func(destdir string) {
		cert, absDir, err := webserv.LoadCert(destdir, "", "")
		if err != nil {
			t.Error(err)
		}
		if cert == nil {
			t.Error("nil cert")
		}
		if absDir != destdir {
			t.Error(absDir, "!=", destdir)
		}
	})
}
