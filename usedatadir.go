package webserv

import (
	"io/fs"
	"os"
	"path/filepath"
)

// DefaultDataDir returns the absolute path to dataDir if not empty, otherwise if
// defaultSuffix is not empty it returns the absolute joined path
// of [os.UserConfigDir] and defaultSuffix.
//
// It will expand environment variables in the path before evaluating the
// absolute path. If expansion collapses the path to empty (for example a lone
// "$VAR" whose variable is unset), the result is empty rather than the current
// working directory.
//
// dataDir and defaultSuffix may contain paths, ".." segments and symlinks.
// They are not confined to the user config directory, so they may resolve
// outside of it. Caller is responsible for validating or sandboxing untrusted
// path input.
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
		// Re-check after expansion: a non-empty input may expand to empty
		// (e.g. "$HOME" with HOME unset), and filepath.Abs("") would resolve
		// to the current working directory rather than leaving result empty.
		if result = os.ExpandEnv(result); result != "" {
			result, err = filepath.Abs(result)
		}
	}
	return
}

// UseDataDir transforms dataDir into an absolute path. Then, if mode
// is not zero, it creates the path if it does not exist using [os.MkdirAll].
// As with [os.MkdirAll], mode is subject to the process umask, so the created
// directory's permissions may be more restrictive than mode. Does nothing if
// dataDir is empty. Does not expand environment variables in the path.
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
