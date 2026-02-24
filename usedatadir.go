package webserv

import (
	"io/fs"
	"os"
	"path/filepath"
)

// DefaultDataDir returns the absolute path to dataDir if not empty, otherwise if
// defaultSuffix is not empty it returns the absolute joined path
// of os.UserConfigDir() and defaultSuffix.
//
// It will expand environment variables in the path before evaluating the
// absolute path.
//
// dataDir and defaultSuffix may contain paths, ".." segments and symlinks.
// They are not confined to UserConfigDir, so they may resolve outside of it.
// Caller is responsible for validating or sandboxing untrusted path input.
func DefaultDataDir(dataDir, defaultSuffix string) (result string, err error) {
	result = dataDir
	if result == "" {
		if defaultSuffix != "" {
			if result, err = os.UserConfigDir(); err == nil {
				result = filepath.Join(result, defaultSuffix)
			}
		}
	}
	if err == nil && result != "" {
		result = os.ExpandEnv(result)
		result, err = filepath.Abs(result)
	}
	return
}

// UseDataDir transforms dataDir into an absolute path. Then, if mode
// is not zero, it creates the path if it does not exist. Does nothing
// if dataDir is empty. Does not expand environment variables in the path.
//
// Returns the final path or an empty string if dataDir was empty.
func UseDataDir(dataDir string, mode fs.FileMode) (string, error) {
	var err error
	if dataDir != "" {
		if dataDir, err = filepath.Abs(dataDir); err == nil {
			if mode != 0 {
				err = os.MkdirAll(dataDir, mode)
			}
		}
	}
	return dataDir, err
}
