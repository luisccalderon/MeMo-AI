//go:build !windows

package pi

import "errors"

// UserEnv has no persistent user scope outside Windows; the process
// environment is the only source there.
func UserEnv(string) (string, bool) {
	return "", false
}

// SetUserEnv is only needed on Windows, where script launchers cannot be
// started without a shell.
func SetUserEnv(string, string) error {
	return errors.New("persistent user environment variables are only managed on Windows")
}
