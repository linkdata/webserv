package webserv

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
)

// DataDirPermissions is the permissions UseDataDir()
// passes on to os.MkdirAll().
var DataDirPermissions = fs.FileMode(0750)

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

func mkdirAll(dataDir string, mode fs.FileMode) (err error) {
	if mode != 0 {
		err = os.MkdirAll(dataDir, mode)
	}
	return
}

// UseDataDir does nothing if dataDir is empty, otherwise it expands
// environment variables and transforms it into an absolute path.
// Then, if mode is not zero, it creates the path if it does not exist.
// Finally, it finally changes the current directory to it.
//
// Returns the final path or an empty string if dataDir was empty.
func UseDataDir(dataDir string, mode fs.FileMode) (string, error) {
	var err error
	if dataDir != "" {
		dataDir = os.ExpandEnv(dataDir)
		if dataDir, err = filepath.Abs(dataDir); err == nil {
			if err = mkdirAll(dataDir, mode); err == nil {
				err = os.Chdir(dataDir)
			}
		}
	}
	return dataDir, err
}
