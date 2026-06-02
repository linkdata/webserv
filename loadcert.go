package webserv

import (
	"crypto/tls"
	"os"
	"path/filepath"
)

// LoadCert does nothing if certDir is empty, otherwise it expands
// environment variables and transforms it into an absolute path.
// It then tries to load a X509 key pair with [crypto/tls.LoadX509KeyPair] from
// the files named fullchainPem and privkeyPem in the resulting directory.
//
// The filenames may contain paths, ".." segments and symlinks.
// They are not confined to certDir, so they may resolve outside of it.
// Caller is responsible for validating or sandboxing untrusted path input.
//
// If fullchainPem is empty, it defaults to [FullchainPem].
// If privkeyPem is empty, it defaults to [PrivkeyPem].
//
// cert is non-nil only when the key pair loaded successfully. absCertDir is the
// resolved absolute directory whenever certDir was non-empty and [path/filepath.Abs]
// succeeded, regardless of whether the key pair then loaded.
func LoadCert(certDir, fullchainPem, privkeyPem string) (cert *tls.Certificate, absCertDir string, err error) {
	if certDir != "" {
		certDir = os.ExpandEnv(certDir)
		if absCertDir, err = filepath.Abs(certDir); err == nil {
			var cer tls.Certificate
			if fullchainPem == "" {
				fullchainPem = FullchainPem
			}
			if privkeyPem == "" {
				privkeyPem = PrivkeyPem
			}
			fc := filepath.Join(absCertDir, fullchainPem)
			pk := filepath.Join(absCertDir, privkeyPem)
			if cer, err = tls.LoadX509KeyPair(fc, pk); err == nil {
				cert = &cer
			}
		}
	}
	return
}
