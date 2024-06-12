package webserv

import "net"

type Config struct {
	Listen    string // optional specific address (and/or port) to listen on
	CertDir   string // if set, directory to look for fullchain.pem and privkey.pem
	User      string // if set, user to switch to after opening listening port
	DataDir   string // if set, ensure this directory exists and switch to it
	ListenURL string // after Apply called, set to an URL we listen on
}

func logInfoQuoted(logger any, msg, val string) {
	if val != "" {
		LogInfo(logger, "%s %q\n", msg, val)
	}
}

// Apply loads certificates if CertDir is set, starts a net.Listener (TLS or normal) on
// the Listen address and port, if User is set it switches to that user and group,
// if DataDir is set, creates that directory if needed and switches to that.
//
// On a non-error return, CertDir and DataDir will be absoulte paths or empty, and
// ListenURL will be a printable and connectable URL like "http://localhost:80".
func (cfg *Config) Apply(logger any) (l net.Listener, err error) {
	if l, cfg.CertDir, cfg.ListenURL, err = Listener(cfg.Listen, cfg.CertDir); err == nil {
		logInfoQuoted(logger, "loaded certificate from", cfg.CertDir)
		if err = BecomeUser(cfg.User); err == nil {
			logInfoQuoted(logger, "switched to user", cfg.User)
			if cfg.DataDir, err = UseDataDir(cfg.DataDir); err == nil {
				logInfoQuoted(logger, "running in data directory", cfg.DataDir)
			}
		}
	}
	return
}
