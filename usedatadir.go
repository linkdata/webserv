package webserv

import (
	"os"
	"path"
	"path/filepath"
)

// DefaultDataDir returns dataDir if not empty, otherwise if
// defaultSuffix is not empty it returns the joined path
// of os.UserConfigDir() and defaultSuffix.
func DefaultDataDir(dataDir, defaultSuffix string) (string, error) {
	var err error
	if dataDir == "" && defaultSuffix != "" {
		dataDir, err = os.UserConfigDir()
		if err == nil {
			dataDir = path.Join(dataDir, defaultSuffix)
		}
	}
	return dataDir, err
}

// UseDataDir expands environment variables in dataDir,
// transforms it into an absolute path, creates it if
// it does not exist and finally changes current directory
// to that path.
//
// Returns the final path.
func UseDataDir(dataDir string) (string, error) {
	var err error
	if dataDir != "" {
		dataDir = os.ExpandEnv(dataDir)
		if dataDir, err = filepath.Abs(dataDir); err == nil {
			if err = os.MkdirAll(dataDir, 0750); err == nil {
				err = os.Chdir(dataDir)
			}
		}
	}
	return dataDir, err
}
