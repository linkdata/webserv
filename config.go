package webserv

import (
	"context"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type Config struct {
	Address              string         // optional specific address (and/or port) to listen on
	CertDir              string         // if set, directory to look for fullchain.pem and privkey.pem
	FullchainPem         string         // set to override filename for "fullchain.pem"
	PrivkeyPem           string         // set to override filename for "privkey.pem"
	User                 string         // if set, user to switch to after opening listening port
	DataDir              string         // if set, change current directory to it
	DataDirMode          fs.FileMode    // if nonzero, create DataDir if it does not exist using this mode
	DefaultDataDirSuffix string         // if set and DataDir is not set, use the user's default data dir plus this suffix
	ListenURL            string         // after Apply called, an URL we listen on (e.g. "https://localhost:8443")
	Logger               InfoLogger     // logger to use, if nil logs nothing
	mu                   sync.Mutex     // protects following
	breakChan            chan os.Signal // break channel
}

func (cfg *Config) BreakChan() (ch chan<- os.Signal) {
	cfg.mu.Lock()
	ch = cfg.breakChan
	cfg.mu.Unlock()
	return
}

func (cfg *Config) logInfo(msg string, keyValuePairs ...any) {
	if cfg.Logger != nil && len(keyValuePairs) > 1 {
		s, ok := keyValuePairs[1].(string)
		if !(ok && s == "") {
			cfg.Logger.Info(msg, keyValuePairs...)
		}
	}
}

// Listen performs initial setup for a simple web server and returns a
// net.Listener if successful.
//
// First it loads certificates if cfg.CertDir is set, and then starts a net.Listener
// (TLS or normal). The listener will default to all addresses and standard port
// depending on privileges and if a certificate was loaded or not.
//
// If cfg.Address was set, any address or port given there overrides these defaults.
//
// If cfg.User is set it then switches to that user and the users primary group.
// Note that this is not supported on Windows.
//
// If cfg.DataDir or cfg.DefaultDataDirSuffix is set, calculates the absolute
// data directory path and sets cfg.DataDir. If cfg.DataDirMode is nonzero, the
// directory will be created if necessary.
//
// On a non-error return, cfg.CertDir and cfg.DataDir will be absolute paths or be empty,
// and cfg.ListenURL will be a printable and connectable URL like "http://localhost:80".
func (cfg *Config) Listen() (l net.Listener, err error) {
	if l, cfg.ListenURL, cfg.CertDir, err = Listener(cfg.Address, cfg.CertDir, cfg.FullchainPem, cfg.PrivkeyPem); err == nil {
		cfg.logInfo("loaded certificates", "dir", cfg.CertDir)
		if err = BecomeUser(cfg.User); err == nil {
			cfg.logInfo("user switched", "user", cfg.User)
			if cfg.DataDir, err = DefaultDataDir(cfg.DataDir, cfg.DefaultDataDirSuffix); err == nil {
				if cfg.DataDir, err = UseDataDir(cfg.DataDir, cfg.DataDirMode); err == nil {
					cfg.logInfo("data directory", "dir", cfg.DataDir)
				}
			}
		}
	}
	return
}

// Serve sets up a signal handler to catch SIGINT and SIGTERM and then
// calls http.Serve.
func (cfg *Config) Serve(ctx context.Context, l net.Listener, handler http.Handler) error {
	breakChan := make(chan os.Signal, 1)
	signal.Notify(breakChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	srv := &http.Server{Handler: handler}
	cfg.logInfo("listening on", "address", l.Addr(), "url", cfg.ListenURL)
	go func() {
		cfg.mu.Lock()
		cfg.breakChan = breakChan
		cfg.mu.Unlock()
		select {
		case sig, ok := <-cfg.breakChan:
			if ok {
				close(breakChan)
				cfg.logInfo("received signal", "sig", sig.String())
			}
		case <-ctx.Done():
		}
		srv.Shutdown(ctx)
	}()
	return srv.Serve(l)
}

// ListenAndServe calls Listen followed by Serve.
func (cfg *Config) ListenAndServe(ctx context.Context, handler http.Handler) (err error) {
	if err = ctx.Err(); err == nil {
		var l net.Listener
		if l, err = cfg.Listen(); err == nil {
			err = cfg.Serve(ctx, l, handler)
		}
	}
	return
}
