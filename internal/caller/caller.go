package caller

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// skipNames are generic wrapper processes to skip when walking the tree.
var skipNames = map[string]bool{
	"go": true, "zsh": true, "bash": true, "sh": true,
	"/bin/zsh": true, "/bin/bash": true, "/bin/sh": true,
	"-/bin/zsh": true, "-/bin/bash": true, // login shells
	"/usr/bin/login": true, "login": true,
	"node": true, // often just a wrapper
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
		name := processName(pid)
		if name == "" {
			break
		}
		if !skipNames[name] {
			return Info{Name: name, PID: pid}
		}
		pid = parentPID(pid)
	}
	// Fallback: return immediate parent
	ppid := os.Getppid()
	return Info{Name: processName(ppid), PID: ppid}
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
