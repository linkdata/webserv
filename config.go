package webserv

type Config struct {
	Listen  string // address:port to listen on
	CertDir string // directory to look for fullchain.pem and privkey.pem
	User    string // user to switch to after opening listening port
	DataDir string // ensure this data directory exists and switch to it
}

func logInfoQuoted(logger any, msg, val string) {
	if val != "" {
		LogInfo(logger, "%s %q", msg, val)
	}
}

func (cfg *Config) Apply(logger any) (l Listener, err error) {
	if l, err = NewListener(cfg.Listen, cfg.CertDir); err == nil {
		logInfoQuoted(logger, "loaded certificate from", l.CertDir)
		if err = BecomeUser(cfg.User); err == nil {
			logInfoQuoted(logger, "switched to user", cfg.User)
			var dataDir string
			if dataDir, err = UseDataDir(dataDir); err == nil {
				cfg.DataDir = dataDir
				logInfoQuoted(logger, "running in data directory", dataDir)
			}
		}
	}
	return
}
