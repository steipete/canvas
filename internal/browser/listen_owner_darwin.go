//go:build darwin

package browser

import (
	"errors"
	"os/exec"
	"strconv"
	"strings"
)

func listeningPIDs(port int) ([]int, error) {
	if port <= 0 {
		return nil, errors.New("missing port")
	}
	out, err := exec.Command("lsof", "-nP", "-iTCP:"+strconv.Itoa(port), "-sTCP:LISTEN", "-Fp").Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, nil
		}
		return nil, err
	}
	return parseLsofListenPIDs(string(out)), nil
}

func parseLsofListenPIDs(out string) []int {
	var pids []int
	seen := map[int]struct{}{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "p") {
			continue
		}
		pid, err := strconv.Atoi(strings.TrimPrefix(line, "p"))
		if err != nil || pid <= 0 {
			continue
		}
		if _, ok := seen[pid]; ok {
			continue
		}
		seen[pid] = struct{}{}
		pids = append(pids, pid)
	}
	return pids
}
