package sources

import (
	"os"
	"path/filepath"
	"strings"

	gossh "github.com/kevinburke/ssh_config"
	"github.com/romain/sshselector/internal/model"
)

// LoadSSHConfig parses ~/.ssh/config and returns a Host for every non-wildcard
// Host block found. Fields missing from the config fall back to ssh defaults
// (empty user, port "22").
func LoadSSHConfig() ([]model.Host, error) {
	path := filepath.Join(os.Getenv("HOME"), ".ssh", "config")

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // not an error — just no config file
		}
		return nil, err
	}
	defer f.Close()

	cfg, err := gossh.Decode(f)
	if err != nil {
		return nil, err
	}

	var hosts []model.Host
	for _, h := range cfg.Hosts {
		// Skip wildcard / default blocks
		if len(h.Patterns) == 0 {
			continue
		}
		alias := h.Patterns[0].String()
		if alias == "*" || strings.ContainsAny(alias, "*?") {
			continue
		}

		hostname := gossh.Get(alias, "HostName")
		if hostname == "" {
			hostname = alias // fall back to the alias itself
		}

		user := gossh.Get(alias, "User")
		port := gossh.Get(alias, "Port")
		if port == "22" {
			port = "" // normalise — we treat empty as "default"
		}

		hosts = append(hosts, model.Host{
			Name:   alias,
			Addr:   hostname,
			User:   user,
			Port:   port,
			Source: model.SourceSSHConfig,
		})
	}

	return hosts, nil
}
