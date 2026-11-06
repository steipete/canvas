//go:build darwin

package browser

import (
	"errors"
	"os/exec"
	"strconv"
	"strings"
)

// lsofPath returns lsof from PATH, or the macOS system binary when PATH
// does not include it. /usr/sbin is not on PATH=/usr/bin:/bin, and a
// LookPath failure is not an lsof exit status.
func lsofPath() string {
	if path, err := exec.LookPath("lsof"); err == nil {
		return path
	}
	return "/usr/sbin/lsof"
}

func listeningPIDs(port int) ([]int, error) {
	if port <= 0 {
		return nil, errors.New("missing port")
	}
	out, err := exec.Command(lsofPath(), "-nP", "-iTCP:"+strconv.Itoa(port), "-sTCP:LISTEN", "-Fp").Output()
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
