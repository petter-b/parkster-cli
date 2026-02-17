package caller

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// skipNames are generic wrapper processes to skip when walking the tree.
// Uses basenames — processName() output is normalized via filepath.Base().
var skipNames = map[string]bool{
	"go": true, "zsh": true, "bash": true, "sh": true,
	"-zsh": true, "-bash": true, "-sh": true, // login shells (prefixed with -)
	"login": true,
	"node":  true, // often just a wrapper
}

// Info holds information about the calling process.
type Info struct {
	Name string // e.g., "claude", "openclaw-gateway"
	PID  int
}

// Detect walks up the process tree from the parent to find the first
// "interesting" ancestor, skipping shells, go tooling, and login wrappers.
// Returns empty Info if detection fails.
func Detect() Info {
	pid := os.Getppid()
	for i := 0; i < 10 && pid > 1; i++ {
		raw := processName(pid)
		if raw == "" {
			break
		}
		name := filepath.Base(raw)
		if !skipNames[name] {
			return Info{Name: name, PID: pid}
		}
		pid = parentPID(pid)
	}
	// Could not find an interesting ancestor
	return Info{}
}

func processName(pid int) string {
	out, err := exec.Command("ps", "-p", fmt.Sprintf("%d", pid), "-o", "comm=").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func parentPID(pid int) int {
	out, err := exec.Command("ps", "-p", fmt.Sprintf("%d", pid), "-o", "ppid=").Output()
	if err != nil {
		return 0
	}
	var ppid int
	_, _ = fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &ppid)
	return ppid
}
