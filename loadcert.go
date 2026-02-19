package webserv

import (
	"crypto/tls"
	"os"
	"path"
	"path/filepath"
)

// LoadCert does nothing if certDir is empty, otherwise it expands
// environment variables and transforms it into an absolute path.
// It then tries to load a X509 key pair from the files named fullchainPem
// and privkeyPem from the resulting directory.
//
// If fullchainPem is empty, it defaults to "fullchain.pem".
// If privkeyPem is empty, it defaults to "privkey.pem".
//
// Return a non-nil cert and absolute path to certDir if there are no errors.
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
			fc := path.Join(absCertDir, path.Base(fullchainPem))
			pk := path.Join(absCertDir, path.Base(privkeyPem))
			if cer, err = tls.LoadX509KeyPair(fc, pk); err == nil {
				cert = &cer
			}
		}
	}
	return
}
