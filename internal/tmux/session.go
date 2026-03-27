package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SessionName derives a tmux session name from a directory path,
// matching the _wt_session_name logic from the shell version.
func SessionName(dir string) string {
	home, _ := os.UserHomeDir()

	devRoot := filepath.Join(home, "dev")
	codeRoot := filepath.Join(home, "code")
	fleetDir := filepath.Join(home, ".config", "fleet")

	if strings.HasPrefix(dir, devRoot+"/") {
		rel := strings.TrimPrefix(dir, devRoot+"/")
		return strings.ReplaceAll(rel, "/", "_")
	}

	if strings.HasPrefix(dir, codeRoot+"/") {
		rel := strings.TrimPrefix(dir, codeRoot+"/")
		return strings.ReplaceAll(rel, "/", "_")
	}

	if dir == fleetDir {
		return "fleet"
	}

	parent := filepath.Base(filepath.Dir(dir))
	base := filepath.Base(dir)
	homeBase := filepath.Base(home)

	if parent == homeBase || parent == "/" {
		return base
	}
	return parent + "." + base
}

// Switch creates or switches to a tmux session for the given target directory.
func Switch(target string) error {
	session := SessionName(target)
	fmt.Fprintf(os.Stderr, "using tmux session %s\n", session)

	inTmux := os.Getenv("TMUX") != ""

	if inTmux {
		if HasSession(session) {
			fmt.Fprintf(os.Stderr, "session %s found, switching...\n", session)
			return SwitchClient(session)
		}
		fmt.Fprintf(os.Stderr, "session %s not found, creating...\n", session)
		if err := NewSessionDetached(session, target); err != nil {
			return err
		}
		return SwitchClient(session)
	}

	fmt.Fprintf(os.Stderr, "session %s not found, starting tmux and creating...\n", session)
	return NewSessionAttach(session, target)
}

// HasSession returns true if a tmux session with the given name exists.
func HasSession(name string) bool {
	return exec.Command("tmux", "has-session", "-t", "="+name).Run() == nil
}

// NewSessionDetached creates a new detached tmux session.
func NewSessionDetached(name, dir string) error {
	return exec.Command("tmux", "new-session", "-d", "-s", name, "-c", dir).Run()
}

// SwitchClient switches the current tmux client to the named session.
func SwitchClient(name string) error {
	return exec.Command("tmux", "switch-client", "-t", "="+name).Run()
}

// NewSessionAttach creates a new session or attaches to an existing one.
func NewSessionAttach(name, dir string) error {
	cmd := exec.Command("tmux", "new-session", "-A", "-s", name, "-c", dir)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ListSessions returns the names of all active tmux sessions.
func ListSessions() []string {
	out, err := exec.Command("tmux", "list-sessions", "-F", "#{session_name}").Output()
	if err != nil {
		return nil
	}
	text := strings.TrimSpace(string(out))
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

// KillSession kills a tmux session by name.
func KillSession(name string) error {
	return exec.Command("tmux", "kill-session", "-t", "="+name).Run()
}

// PaneProcesses returns the processes running in a tmux session's panes.
func PaneProcesses(session string) []string {
	if !HasSession(session) {
		return nil
	}
	out, err := exec.Command("tmux", "list-panes", "-t", "="+session, "-F", "#{pane_pid}").Output()
	if err != nil {
		return nil
	}

	pids := strings.Split(strings.TrimSpace(string(out)), "\n")
	var procs []string
	for _, pid := range pids {
		pid = strings.TrimSpace(pid)
		if pid == "" {
			continue
		}
		childOut, err := exec.Command("pgrep", "-P", pid).Output()
		if err != nil {
			continue
		}
		for _, cpid := range strings.Split(strings.TrimSpace(string(childOut)), "\n") {
			cpid = strings.TrimSpace(cpid)
			if cpid == "" {
				continue
			}
			psOut, err := exec.Command("ps", "-p", cpid, "-o", "pid=,comm=,args=").Output()
			if err != nil {
				continue
			}
			info := strings.TrimSpace(string(psOut))
			if info != "" {
				procs = append(procs, info)
			}
		}
	}
	return procs
}
