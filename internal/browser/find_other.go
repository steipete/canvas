//go:build !darwin && !linux

package browser

import "errors"

func FindChromiumBinary() (string, error) {
	return "", errors.New("FindChromiumBinary is only implemented on darwin (set --browser-bin)")
}
