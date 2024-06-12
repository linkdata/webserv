package webserv

import (
	"crypto/tls"
	"path"
	"path/filepath"
)

// LoadCert does nothing if certDir is empty, otherwise tries to load a X509 key pair
// from files named "fullchain.pem" and "privkey.pem" in the given directory.
//
// Return a non-nil cert and absolute path to certDir if there are no errors.
func LoadCert(certDir string) (cert *tls.Certificate, absCertDir string, err error) {
	if certDir != "" {
		if absCertDir, err = filepath.Abs(certDir); err == nil {
			var cer tls.Certificate
			fc := path.Join(absCertDir, FullchainPem)
			pk := path.Join(absCertDir, PrivkeyPem)
			if cer, err = tls.LoadX509KeyPair(fc, pk); err == nil {
				cert = &cer
			}
		}
	}
	return
}
