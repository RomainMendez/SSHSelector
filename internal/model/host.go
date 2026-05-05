package model

// Source identifies where a Host entry came from.
type Source string

const (
	SourceSSHConfig Source = "ssh_config"
	SourceTailscale Source = "tailscale"
)

// Host represents a single SSH target.
type Host struct {
	// Name is the display label (SSH alias or Tailscale MagicDNS hostname).
	Name string
	// Addr is the address passed to ssh (hostname, IP, or MagicDNS name).
	Addr string
	// User is the SSH username. Empty means use the system/ssh default.
	User string
	// Port is the SSH port. Empty or "22" means default.
	Port string
	// Source indicates which provider supplied this host.
	Source Source
	// Tags holds optional key=value metadata (e.g. Tailscale tags).
	Tags []string
}

// SSHTarget returns the user@addr string suitable for the ssh command.
func (h Host) SSHTarget() string {
	if h.User != "" {
		return h.User + "@" + h.Addr
	}
	return h.Addr
}
