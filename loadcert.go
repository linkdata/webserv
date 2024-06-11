package webserv

import (
	"crypto/tls"
	"path"
	"path/filepath"
)

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
