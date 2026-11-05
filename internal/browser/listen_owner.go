package browser

import (
	"errors"
	"os/exec"
	"strconv"
	"strings"
)

// procNetListenInode parses one /proc/net/tcp or tcp6 line.
// It returns the socket inode when the line is a listener on port.
func procNetListenInode(line string, port int) (string, bool) {
	fields := strings.Fields(line)
	// sl local rem st tx:rx tr:when retrnsmt uid timeout inode
	if len(fields) < 10 || !strings.EqualFold(fields[3], "0A") {
		return "", false
	}
	parts := strings.Split(fields[1], ":")
	if len(parts) != 2 {
		return "", false
	}
	parsed, err := strconv.ParseUint(parts[1], 16, 16)
	if err != nil || int(parsed) != port {
		return "", false
	}
	if fields[9] == "" || fields[9] == "0" {
		return "", false
	}
	return fields[9], true
}

func listenerOwnedBy(port, pid int) bool {
	owned, _ := listenerOwnership(port, pid)
	return owned
}

func listenerOwnership(port, pid int) (owned bool, foreign int) {
	if port <= 0 || pid <= 0 {
		return false, 0
	}
	pids, err := listeningPIDs(port)
	if err != nil || len(pids) == 0 {
		return false, 0
	}
	for _, lp := range pids {
		if processInTree(lp, pid) {
			return true, 0
		}
	}
	return false, pids[0]
}

func processInTree(pid, root int) bool {
	if pid <= 0 || root <= 0 {
		return false
	}
	seen := map[int]struct{}{}
	for i := 0; i < 12 && pid > 1; i++ {
		if pid == root {
			return true
		}
		if _, ok := seen[pid]; ok {
			return false
		}
		seen[pid] = struct{}{}
		next, err := parentPID(pid)
		if err != nil || next == pid || next <= 0 {
			return false
		}
		pid = next
	}
	return pid == root
}

func parentPID(pid int) (int, error) {
	out, err := exec.Command("ps", "-o", "ppid=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return 0, err
	}
	parent, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil || parent <= 0 {
		return 0, errors.New("missing parent pid")
	}
	return parent, nil
}

func browserProcessExited(pid int) bool {
	if pid <= 0 {
		return true
	}
	out, err := exec.Command("ps", "-o", "stat=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		var exitErr *exec.ExitError
		return errors.As(err, &exitErr)
	}
	stat := strings.TrimSpace(string(out))
	return stat == "" || strings.HasPrefix(stat, "Z")
}
