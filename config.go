package webserv

import (
	"log/slog"
	"net"
)

type Config struct {
	Listen               string // optional specific address (and/or port) to listen on
	CertDir              string // if set, directory to look for fullchain.pem and privkey.pem
	User                 string // if set, user to switch to after opening listening port
	DataDir              string // if set, ensure this directory exists and switch to it
	DefaultDataDirSuffix string // if set and DataDir is not set, use the user's default data dir plus this suffix
	ListenURL            string // after Apply called, an URL we listen on (e.g. "https://localhost:8443")
}

func logInfo(logger *slog.Logger, msg, key, val string) {
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
// If DataDir or DefaultDataDirSuffix is set, creates that directory if needed
// and sets the current working directory to it.
//
// On a non-error return, CertDir and DataDir will be absolute paths or empty, and
// ListenURL will be a printable and connectable URL like "http://localhost:80".
func (cfg *Config) Apply(logger *slog.Logger) (l net.Listener, err error) {
	if l, cfg.ListenURL, cfg.CertDir, err = Listener(cfg.Listen, cfg.CertDir); err == nil {
		logInfo(logger, "loaded certificates", "dir", cfg.CertDir)
		if err = BecomeUser(cfg.User); err == nil {
			logInfo(logger, "user switched", "user", cfg.User)
			if cfg.DataDir, err = DefaultDataDir(cfg.DataDir, cfg.DefaultDataDirSuffix); err == nil {
				if cfg.DataDir, err = UseDataDir(cfg.DataDir); err == nil {
					logInfo(logger, "using data directory", "dir", cfg.DataDir)
				}
			}
		}
	}
	return
}
