# SSHSelector

A terminal UI for quickly SSHing into hosts from your `~/.ssh/config` and your Tailscale network. Pick a host, confirm the user and port, and you are dropped straight into the SSH session — no typing long addresses or remembering IP addresses.

---

## Features

- **`~/.ssh/config` support** — every non-wildcard `Host` block is surfaced automatically, with its `HostName`, `User`, and `Port` pre-filled
- **Tailscale peer discovery** — runs `tailscale status --json` locally; no API token required
- **WSL2 interop** — when running inside WSL2, falls back to `tailscale.exe` on the Windows host automatically
- **Fuzzy filtering** — type to narrow the list in real time across host name, address, and username
- **Two-screen flow** — pick a host, then review and edit the user and port before connecting
- **Live command preview** — the exact `ssh` command is shown before you press Enter
- **Clean handoff** — uses `syscall.Exec` to replace the process with SSH; no wrapper left behind, SSH owns the terminal fully
- **Source badges** — hosts are labelled `[ssh]` or `[ts]` so you always know where they came from

---

## Requirements

| Dependency | Notes |
|---|---|
| Go 1.21+ | For building |
| OpenSSH | Must be at `/usr/bin/ssh`, `/bin/ssh`, or `/usr/local/bin/ssh` |
| Tailscale | Optional — native Linux daemon, or Windows client via WSL2 interop |

---

## Building

```bash
git clone https://github.com/romain/sshselector
cd sshselector
```

To build **and** install system-wide in one step, run the provided install script:

```bash
./install.sh
```

The script will:
1. Verify you are in the correct repository root
2. Build a stripped binary with version info embedded (`git describe`)
3. Move it to `/usr/local/bin/sshselector`, prompting for `sudo` if needed
4. Confirm the binary is on your `$PATH`

To build without installing:

```bash
go build -o sshselector .
```

---

## Usage

```bash
sshselector
```

No flags or arguments — everything is discovered automatically.

### Screen 1 — Host picker

```
  SSH Selector
  Type to filter hosts…

▶ [ts]  my-server    100.64.0.5
  [ts]  dev-box      100.64.0.9
  [ssh] homelab      192.168.1.10

  ↑/↓ navigate   enter select   esc quit
```

| Key | Action |
|---|---|
| `↑` / `↓` | Move through the list |
| `Ctrl+P` / `Ctrl+N` | Same as ↑ / ↓ |
| Type anything | Filter the list |
| `Enter` | Select the highlighted host |
| `Esc` / `Ctrl+C` | Quit |

### Screen 2 — Connection details

```
  Connect to host
  [ts]  my-server  100.64.0.5

  User    romain
  Port    22

  Command: ssh romain@100.64.0.5

  tab cycle fields   enter connect   esc back
```

The `User` field is pre-filled from the host config, falling back to `$USER` if none is set. The `Port` field defaults to `22`. Edit either value freely before connecting.

| Key | Action |
|---|---|
| `Tab` / `Shift+Tab` | Cycle between User and Port fields |
| `Enter` | Connect with the current values |
| `Esc` | Go back to the host list |
| `Ctrl+C` | Quit entirely |

---

## Host sources

### `~/.ssh/config`

Every `Host` block that is not a wildcard (`*`) is loaded. The following directives are read:

| SSH directive | Maps to |
|---|---|
| `Host` | Display name |
| `HostName` | Address passed to `ssh` (falls back to the alias if absent) |
| `User` | SSH username |
| `Port` | SSH port |

Example `~/.ssh/config` entry:

```ssh-config
Host homelab
    HostName 192.168.1.10
    User romain
    Port 2222
```

### Tailscale

`tailscale status --json` is called at startup. For every peer (and the local machine itself) the following fields are extracted:

| Tailscale field | Used for |
|---|---|
| `DNSName` | Address (MagicDNS name, preferred — trailing dot stripped) |
| `HostName` | Display name |
| `TailscaleIPs[0]` | Address fallback if MagicDNS is unavailable |
| `Tags` | Stored on the host for future use |

No user is pre-filled for Tailscale hosts — the detail screen defaults to `$USER` and lets you change it before connecting.

Hosts that appear in both sources (same address) are deduplicated — the `~/.ssh/config` entry takes precedence.

---

## WSL2 + Tailscale

Tailscale typically runs on the **Windows host** when working inside WSL2. SSHSelector handles this in two ways:

### Network reachability

WSL2 shares the Windows network stack. Tailscale IPs are routed by the Windows Tailscale client and are directly reachable from inside WSL2 — no extra configuration needed.

### Binary discovery

SSHSelector resolves the Tailscale binary in order, stopping at the first match:

| Priority | Path tried | Scenario |
|---|---|---|
| 1 | `tailscale` (via `$PATH`) | Native Linux Tailscale installed inside WSL2 |
| 2 | `tailscale.exe` (via `$PATH`) | WSL2 interop — Windows binary surfaced on `$PATH` |
| 3 | `/mnt/c/Program Files/Tailscale/tailscale.exe` | Hard fallback if `appendWindowsPath` is disabled |

If none of the above is found, Tailscale discovery is skipped silently and only `~/.ssh/config` hosts are shown.

### WSL2 interop requirements

Interop is enabled by default. If you have a custom `/etc/wsl.conf`, make sure it is not disabled:

```ini
[interop]
enabled = true
appendWindowsPath = true
```

If `appendWindowsPath` is `false`, the hard-coded `/mnt/c/Program Files/Tailscale/tailscale.exe` fallback will still be tried.

---

## Project structure

```
SSHSelector/
├── main.go                       # Entry point — loads sources, runs TUI, execs SSH
├── go.mod
├── go.sum
└── internal/
    ├── model/
    │   └── host.go               # Shared Host struct
    ├── sources/
    │   ├── sshconfig.go          # Parses ~/.ssh/config
    │   └── tailscale.go          # Queries tailscale status --json
    └── tui/
        └── tui.go                # Bubbletea TUI (list screen + detail screen)
```

---

## License

MIT — see [LICENSE](LICENSE).
