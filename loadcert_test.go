package webserv_test

import (
	"testing"

	"github.com/linkdata/webserv"
)

func TestLoadCert(t *testing.T) {
	cert, absDir, err := webserv.LoadCert("")
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
		cert, absDir, err := webserv.LoadCert(destdir)
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
