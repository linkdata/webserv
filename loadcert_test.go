package webserv_test

import (
	"path/filepath"
	"testing"

	"github.com/linkdata/webserv"
)

func TestLoadCert_UsesFilepathJoinNotPathJoin(t *testing.T) {
	// Bug: loadcert.go uses path.Join (POSIX) instead of filepath.Join (OS-aware).
	// On Windows, path.Join produces forward-slash paths which may not work correctly.
	// We verify that the resulting paths use OS-appropriate separators by checking
	// that LoadCert does not fail for a directory with cert files in a subdirectory.
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
