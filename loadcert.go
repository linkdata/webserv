package webserv

import (
	"crypto/tls"
	"path"
	"path/filepath"
)

// LoadCert does nothing if certDir is empty, otherwise tries to load a X509 key pair
// from files named fullchainPem and privkeyPem in the given directory certDir.
//
// If fullchainPem is empty, it defaults to "fullchain.pem".
// If privkeyPem is empty, it defaults to "privkey.pem".
//
// Return a non-nil cert and absolute path to certDir if there are no errors.
func LoadCert(certDir, fullchainPem, privkeyPem string) (cert *tls.Certificate, absCertDir string, err error) {
	if certDir != "" {
		if absCertDir, err = filepath.Abs(certDir); err == nil {
			var cer tls.Certificate
			if fullchainPem == "" {
				fullchainPem = FullchainPem
			}
			if privkeyPem == "" {
				privkeyPem = PrivkeyPem
			}
			fc := path.Join(absCertDir, fullchainPem)
			pk := path.Join(absCertDir, privkeyPem)
			if cer, err = tls.LoadX509KeyPair(fc, pk); err == nil {
				cert = &cer
			}
		}
	}
	return
}
