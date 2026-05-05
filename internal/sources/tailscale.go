package sources

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/romain/sshselector/internal/model"
)

// tailscaleStatus is the top-level JSON structure returned by
// `tailscale status --json`.
type tailscaleStatus struct {
	Self  tailscalePeer            `json:"Self"`
	Peers map[string]tailscalePeer `json:"Peer"`
}

type tailscalePeer struct {
	HostName     string   `json:"HostName"`
	DNSName      string   `json:"DNSName"`
	TailscaleIPs []string `json:"TailscaleIPs"`
	UserID       int64    `json:"UserID"`
	Tags         []string `json:"Tags"`
	Online       bool     `json:"Online"`
}

// findTailscaleBin resolves the tailscale binary using a priority chain that
// works for native Linux, WSL2 (via Windows interop), and a hard-coded
// fallback to the default Windows installation path mounted at /mnt/c.
func findTailscaleBin() (string, bool) {
	// 1. Native Linux tailscale in $PATH.
	if p, err := exec.LookPath("tailscale"); err == nil {
		return p, true
	}
	// 2. Windows tailscale.exe via WSL2 interop (usually on $PATH automatically).
	if p, err := exec.LookPath("tailscale.exe"); err == nil {
		return p, true
	}
	// 3. Hard fallback: default Tailscale install location on Windows C: drive,
	//    which WSL2 mounts at /mnt/c.
	const windowsFallback = "/mnt/c/Program Files/Tailscale/tailscale.exe"
	if _, err := exec.LookPath(windowsFallback); err == nil {
		return windowsFallback, true
	}
	// Try stat directly — LookPath requires the file to be executable-flagged,
	// which Windows binaries mounted via WSL2 always are.
	if info, err := exec.Command("test", "-f", windowsFallback).Output(); err == nil || info != nil {
		// Use the path directly and let exec sort it out.
		return windowsFallback, true
	}
	return "", false
}

// LoadTailscaleHosts runs `tailscale status --json` and returns a Host for
// every peer (and self) visible on the tailnet.
// If the tailscale binary is not found or the tailnet is not connected,
// an empty slice is returned without error so the rest of the app still works.
// In WSL2 environments, it automatically falls back to tailscale.exe running
// on the Windows host via WSL2 interop.
func LoadTailscaleHosts() ([]model.Host, error) {
	path, ok := findTailscaleBin()
	if !ok {
		// Neither Linux tailscale nor Windows tailscale.exe found — skip silently.
		return nil, nil
	}

	out, err := exec.Command(path, "status", "--json").Output() //nolint:gosec
	if err != nil {
		// Not connected, daemon not running, etc. — silently skip.
		return nil, nil
	}

	var status tailscaleStatus
	if err := json.Unmarshal(out, &status); err != nil {
		return nil, fmt.Errorf("tailscale: parse status JSON: %w", err)
	}

	var hosts []model.Host

	// Include self so the user can SSH into the local node's Tailscale address.
	if self := peerToHost(status.Self, true); self != nil {
		hosts = append(hosts, *self)
	}

	for _, peer := range status.Peers {
		if h := peerToHost(peer, false); h != nil {
			hosts = append(hosts, *h)
		}
	}

	return hosts, nil
}

func peerToHost(p tailscalePeer, isSelf bool) *model.Host {
	// Prefer MagicDNS name (strip trailing dot), fall back to raw hostname.
	addr := strings.TrimSuffix(p.DNSName, ".")
	name := p.HostName
	if addr == "" {
		if len(p.TailscaleIPs) > 0 {
			addr = p.TailscaleIPs[0]
		} else {
			return nil // no usable address
		}
	}
	if name == "" {
		name = addr
	}

	label := name
	if isSelf {
		label = name + " (this machine)"
	}

	return &model.Host{
		Name:   label,
		Addr:   addr,
		Source: model.SourceTailscale,
		Tags:   p.Tags,
	}
}
