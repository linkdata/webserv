package webserv

import (
	"io/fs"
	"net"
)

type Config struct {
	Listen               string      // optional specific address (and/or port) to listen on
	CertDir              string      // if set, directory to look for fullchain.pem and privkey.pem
	FullchainPem         string      // set to override filename for "fullchain.pem"
	PrivkeyPem           string      // set to override filename for "privkey.pem"
	User                 string      // if set, user to switch to after opening listening port
	DataDir              string      // if set, change current directory to it
	DataDirMode          fs.FileMode // if nonzero, create DataDir if it does not exist using this mode
	DefaultDataDirSuffix string      // if set and DataDir is not set, use the user's default data dir plus this suffix
	ListenURL            string      // after Apply called, an URL we listen on (e.g. "https://localhost:8443")
}

func logInfo(logger InfoLogger, msg, key, val string) {
	if logger != nil && val != "" {
		logger.Info(msg, key, val)
	}
}

// Apply performs initial setup for a simple web server, optionally
// logging informational messages if it loads certificates, switches
// the current user, or switches to the data directory.
//
// First it loads certificates if CertDir is set, and then starts a net.Listener
// (TLS or normal). The listener will default to all addresses and standard port
// depending on privileges and if a certificate was loaded or not.
//
// If Listen was set, any address or port given there overrides these defaults.
//
// If User is set it then switches to that user and the users primary group.
// Note that this is not supported on Windows.
//
// If DataDir or DefaultDataDirSuffix is set, changes the current working
// directory. If DataDirMode is nonzero, the directory will be created
// if necessary.
//
// On a non-error return, CertDir and DataDir will be absolute paths or be empty,
// and ListenURL will be a printable and connectable URL like "http://localhost:80".
func (cfg *Config) Apply(logger InfoLogger) (l net.Listener, err error) {
	if l, cfg.ListenURL, cfg.CertDir, err = Listener(cfg.Listen, cfg.CertDir, cfg.FullchainPem, cfg.PrivkeyPem); err == nil {
		logInfo(logger, "loaded certificates", "dir", cfg.CertDir)
		if err = BecomeUser(cfg.User); err == nil {
			logInfo(logger, "user switched", "user", cfg.User)
			if cfg.DataDir, err = DefaultDataDir(cfg.DataDir, cfg.DefaultDataDirSuffix); err == nil {
				if cfg.DataDir, err = UseDataDir(cfg.DataDir, cfg.DataDirMode); err == nil {
					logInfo(logger, "using data directory", "dir", cfg.DataDir)
				}
			}
		}
	}
	return
}
