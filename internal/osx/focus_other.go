//go:build !darwin

package osx

import "errors"

func FocusPID(pid int) error {
	return errors.New("focus is only supported on macOS")
}
