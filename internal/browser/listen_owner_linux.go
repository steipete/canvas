//go:build linux

package browser

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func listeningPIDs(port int) ([]int, error) {
	if port <= 0 {
		return nil, errors.New("missing port")
	}
	inodes, err := listenInodes(port)
	if err != nil || len(inodes) == 0 {
		return nil, err
	}
	return pidsForSocketInodes(inodes)
}

func listenInodes(port int) (map[string]struct{}, error) {
	inodes := map[string]struct{}{}
	for _, path := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		err := collectListenInodes(path, port, inodes)
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}
	return inodes, nil
}

func collectListenInodes(path string, port int, inodes map[string]struct{}) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	first := true
	for sc.Scan() {
		if first {
			first = false
			continue
		}
		inode, ok := procNetListenInode(sc.Text(), port)
		if !ok {
			continue
		}
		inodes[inode] = struct{}{}
	}
	return sc.Err()
}

func pidsForSocketInodes(inodes map[string]struct{}) ([]int, error) {
	proc, err := os.Open("/proc")
	if err != nil {
		return nil, err
	}
	defer proc.Close()
	names, err := proc.Readdirnames(-1)
	if err != nil {
		return nil, err
	}

	var pids []int
	for _, name := range names {
		pid, err := strconv.Atoi(name)
		if err != nil || pid <= 0 {
			continue
		}
		if processHasSocketInode(pid, inodes) {
			pids = append(pids, pid)
		}
	}
	return pids, nil
}

func processHasSocketInode(pid int, inodes map[string]struct{}) bool {
	fdDir := fmt.Sprintf("/proc/%d/fd", pid)
	entries, err := os.ReadDir(fdDir)
	if err != nil {
		return false
	}
	for _, ent := range entries {
		target, err := os.Readlink(filepath.Join(fdDir, ent.Name()))
		if err != nil {
			continue
		}
		if !strings.HasPrefix(target, "socket:[") || !strings.HasSuffix(target, "]") {
			continue
		}
		inode := strings.TrimSuffix(strings.TrimPrefix(target, "socket:["), "]")
		if _, ok := inodes[inode]; ok {
			return true
		}
	}
	return false
}
