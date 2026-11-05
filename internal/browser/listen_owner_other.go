//go:build !darwin && !linux

package browser

import "errors"

func listeningPIDs(port int) ([]int, error) {
	return nil, errors.New("devtools listener ownership is only implemented on darwin and linux")
}
