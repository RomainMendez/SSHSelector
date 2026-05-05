package main

import (
	"fmt"
	"os"
	"syscall"

	"github.com/romain/sshselector/internal/model"
	"github.com/romain/sshselector/internal/sources"
	"github.com/romain/sshselector/internal/tui"
)

func main() {
	// ------------------------------------------------------------------ //
	// 1. Collect hosts from all sources
	// ------------------------------------------------------------------ //

	var allHosts []model.Host
	seen := make(map[string]struct{}) // deduplicate by addr

	sshHosts, err := sources.LoadSSHConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not parse ~/.ssh/config: %v\n", err)
	}
	for _, h := range sshHosts {
		key := h.Addr
		if _, dup := seen[key]; !dup {
			seen[key] = struct{}{}
			allHosts = append(allHosts, h)
		}
	}

	tsHosts, err := sources.LoadTailscaleHosts()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not query tailscale: %v\n", err)
	}
	for _, h := range tsHosts {
		key := h.Addr
		if _, dup := seen[key]; !dup {
			seen[key] = struct{}{}
			allHosts = append(allHosts, h)
		}
	}

	// ------------------------------------------------------------------ //
	// 2. Run TUI
	// ------------------------------------------------------------------ //

	chosen, err := tui.Run(allHosts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if chosen == nil {
		// User pressed Escape / Ctrl+C — exit cleanly.
		os.Exit(0)
	}

	// ------------------------------------------------------------------ //
	// 3. Hand off to SSH via exec(2) so the session owns the terminal fully
	// ------------------------------------------------------------------ //

	sshBin, err := findSSH()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	args := buildSSHArgs(sshBin, *chosen)

	// Print the command so the user can see what is being run.
	fmt.Fprintf(os.Stderr, "Connecting: %s\n", joinArgs(args))

	if err := syscall.Exec(sshBin, args, os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "exec ssh: %v\n", err)
		os.Exit(1)
	}
}

// findSSH locates the ssh binary, trying common paths.
func findSSH() (string, error) {
	candidates := []string{
		"/usr/bin/ssh",
		"/bin/ssh",
		"/usr/local/bin/ssh",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("ssh binary not found; is OpenSSH installed?")
}

// buildSSHArgs constructs the argv slice for the ssh invocation.
func buildSSHArgs(bin string, h model.Host) []string {
	args := []string{bin}
	if h.Port != "" && h.Port != "22" {
		args = append(args, "-p", h.Port)
	}
	args = append(args, h.SSHTarget())
	return args
}

func joinArgs(args []string) string {
	out := ""
	for i, a := range args {
		if i > 0 {
			out += " "
		}
		out += a
	}
	return out
}
