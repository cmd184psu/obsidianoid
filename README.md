# Obsidianoid

A lightweight, self-hosted Obsidian vault viewer and editor. Browse, read, and
edit your Obsidian markdown notes from any browser on your local network —
no Obsidian installation required.

**Features**
- Dark-themed two-panel UI (resizable sidebar + content area)
- File tree with live search/filter
- Edit notes in a monospace textarea
- Toggle between raw markdown editor and rendered GitHub-flavored HTML preview
- Save notes back to disk (`Ctrl/Cmd+S` or the Save button)
- New note creation
- HTTPS with self-signed or real certificates
- Runs as a systemd service on Linux / Raspberry Pi

---

## Directory Layout

```
/opt/obsidianoid/
  static/           ← HTML, CSS, JS served to the browser
    index.html
    css/app.css
    js/app.js

/usr/local/bin/obsidianoid   ← compiled binary

~/.obsidianoid.json          ← runtime config (per-user)
~/.obsidianoid/
  server.crt                 ← TLS certificate
  server.key                 ← TLS private key
```

---

## Requirements

| Tool | Purpose |
|------|---------|
| Go 1.21+ | Build the binary |
| make | Run Makefile targets |
| systemd | Service management (Linux) |

---

## Building

### On the same machine (Linux/macOS)

```bash
cd /opt/obsidianoid
go mod tidy
make build
```

### Cross-compile for Raspberry Pi (arm64 — Pi 3B+, 4, 5)

```bash
make build-pi
# produces: ./obsidianoid-arm64
```

### Cross-compile for older Pi models (armv7 — Pi 2, 3 32-bit OS)

```bash
make build-pi-armv7
# produces: ./obsidianoid-armv7
```

Copy the binary to the Pi:

```bash
scp obsidianoid-arm64 pi@raspberrypi.local:/tmp/obsidianoid
ssh pi@raspberrypi.local "sudo mv /tmp/obsidianoid /usr/local/bin/obsidianoid && sudo chmod 755 /usr/local/bin/obsidianoid"
```

---

## TLS Certificates

All cert files live in `~/.obsidianoid/` (the home directory of the user running
the service).

```
~/.obsidianoid/
  server.crt    ← certificate (PEM)
  server.key    ← private key  (PEM, chmod 600)
```

### Option A — Self-signed (quick start)

```bash
obsidianoid -gen-cert
# Writes ~/.obsidianoid/server.crt and ~/.obsidianoid/server.key
# Valid for 10 years, covers localhost + 127.0.0.1 + ::1
```

On first browser visit accept the certificate warning, or trust it permanently:

- **macOS**: open `~/.obsidianoid/server.crt` in Keychain Access → set to *Always Trust*
- **iOS/iPadOS**: AirDrop the `.crt` to the device → Settings → General →
  Profile Downloaded → Install, then Settings → General → About →
  Certificate Trust Settings → toggle on
- **Linux**: `sudo cp ~/.obsidianoid/server.crt /usr/local/share/ca-certificates/obsidianoid.crt && sudo update-ca-certificates`

### Option B — Real certificate (Let’s Encrypt / your own CA)

Override the default cert paths in `~/.obsidianoid.json`:

```json
{
  "vault_path": "/home/pi/vault",
  "port": 8989,
  "cert_file": "/etc/letsencrypt/live/yourhost.example.com/fullchain.pem",
  "key_file":  "/etc/letsencrypt/live/yourhost.example.com/privkey.pem"
}
```

If you are terminating TLS at HAProxy and running obsidianoid plain HTTP
behind it, use the `-insecure` flag (see HAProxy section below).

---

## Configuration

The config file lives at `~/.obsidianoid.json` (relative to the service user's
home directory).

```json
{
  "vault_path": "/home/pi/vault",
  "port": 8989,
  "cert_file": "/home/pi/.obsidianoid/server.crt",
  "key_file":  "/home/pi/.obsidianoid/server.key",
  "custom_css": "/home/pi/.obsidianoid/custom.css"
}
```

| Key | Required | Default | Description |
|-----|----------|---------|-------------|
| `vault_path` | **yes** | — | Absolute path to your Obsidian vault directory |
| `port` | no | `8989` | TCP port to listen on |
| `cert_file` | no | `~/.obsidianoid/server.crt` | Path to PEM certificate |
| `key_file` | no | `~/.obsidianoid/server.key` | Path to PEM private key |
| `custom_css` | no | — | Path to a CSS file that overrides the embedded markdown preview styles |

### custom_css

The CSS file pointed to by `custom_css` is injected into the markdown preview
pane after the built-in GitHub-flavored styles. Use it to override fonts,
colours, spacing, etc. without touching the app source. Example:

```css
/* ~/.obsidianoid/custom.css */
.md-body {
  font-family: 'Georgia', serif;
  font-size: 1.05rem;
  max-width: 80ch;
}
.md-body code {
  background: #1e1e2e;
}
```

---

## Install as a systemd Service

### 1. Create a dedicated service user (recommended)

```bash
sudo useradd -r -s /bin/false -m -d /home/obsidianoid obsidianoid
```

Place the config and certs under that user's home:

```bash
sudo -u obsidianoid mkdir -p /home/obsidianoid/.obsidianoid
# Copy or generate certs:
sudo -u obsidianoid /usr/local/bin/obsidianoid -gen-cert
# Create config:
sudo -u obsidianoid tee /home/obsidianoid/.obsidianoid.json <<'EOF'
{
  "vault_path": "/path/to/your/vault",
  "port": 8989
}
EOF
```

Make sure the vault directory is readable by the service user:

```bash
sudo chown -R obsidianoid:obsidianoid /path/to/your/vault
# or just add read permission:
sudo chmod -R o+r /path/to/your/vault
```

### 2. Install with make

Run from the project root on the target machine (or after copying files over):

```bash
sudo make install
```

This will:
1. Build the binary
2. Copy it to `/usr/local/bin/obsidianoid` (chmod 755)
3. Copy `static/` to `/opt/obsidianoid/static/`
4. Install `obsidianoid.service` to `/etc/systemd/system/`
5. Run `systemctl daemon-reload && systemctl enable obsidianoid && systemctl start obsidianoid`

### 3. Useful commands

```bash
sudo make status    # systemctl status obsidianoid
sudo make logs      # journalctl -u obsidianoid -f
sudo make restart   # after config changes
sudo make stop
sudo make start
sudo make uninstall # removes service + binary (leaves config/certs/vault)
```

### 4. Run tests

```bash
make test           # green PASS / red FAIL output
```

---

## Running Manually

```bash
# HTTPS (default)
obsidianoid

# Plain HTTP (no certs needed — LAN only)
obsidianoid -insecure

# Custom config path
obsidianoid -config /etc/obsidianoid.json

# Generate self-signed cert and exit
obsidianoid -gen-cert
```

---

## HAProxy Integration

If you already run HAProxy on port 443 and want obsidianoid available at
`https://yourpi.local/notes` (or a dedicated subdomain), terminate TLS at
HAProxy and proxy plain HTTP to obsidianoid running with `-insecure`.

### Option A — Path-based routing (`/notes`)

```haproxy
#--------------------------------------------------------------------
# /etc/haproxy/haproxy.cfg  (relevant sections)
#--------------------------------------------------------------------
global
    log /dev/log local0
    maxconn 2048

defaults
    log     global
    mode    http
    option  httplog
    option  dontlognull
    timeout connect 5s
    timeout client  60s
    timeout server  60s

frontend https_in
    bind *:443 ssl crt /etc/haproxy/certs/yourpi.pem alpn h2,http/1.1
    bind *:80
    http-request redirect scheme https unless { ssl_fc }

    # Route /notes and /notes/* to obsidianoid
    acl is_notes path_beg /notes
    use_backend obsidianoid_back if is_notes

    # Everything else goes to your default backend
    default_backend main_back

backend obsidianoid_back
    # Strip the /notes prefix before forwarding
    http-request replace-path /notes(.*) \1
    http-request set-header X-Forwarded-Proto https
    server obsidianoid 127.0.0.1:8989 check

backend main_back
    server main 127.0.0.1:80 check
```

Run obsidianoid without TLS:

```bash
obsidianoid -insecure
# or in the service file, change ExecStart to:
# ExecStart=/usr/local/bin/obsidianoid -insecure
```

### Option B — Subdomain routing (`notes.yourpi.local`)

```haproxy
frontend https_in
    bind *:443 ssl crt /etc/haproxy/certs/yourpi.pem alpn h2,http/1.1

    acl is_notes_subdomain hdr(host) -i notes.yourpi.local
    use_backend obsidianoid_back if is_notes_subdomain

    default_backend main_back

backend obsidianoid_back
    http-request set-header X-Forwarded-Proto https
    server obsidianoid 127.0.0.1:8989 check
```

### HAProxy certificate format

HAProxy expects a single `.pem` file that is the certificate + private key
concatenated:

```bash
cat /etc/letsencrypt/live/yourpi.example.com/fullchain.pem \
    /etc/letsencrypt/live/yourpi.example.com/privkey.pem \
    > /etc/haproxy/certs/yourpi.pem
chmod 600 /etc/haproxy/certs/yourpi.pem
```

For a self-signed cert:

```bash
cat ~/.obsidianoid/server.crt ~/.obsidianoid/server.key \
    > /etc/haproxy/certs/obsidianoid.pem
chmod 600 /etc/haproxy/certs/obsidianoid.pem
```

---

## Firewall

### Raspberry Pi (iptables / nftables)

```bash
# Allow port 8989 from LAN only (adjust subnet to match yours)
sudo iptables -A INPUT -p tcp --dport 8989 -s 192.168.1.0/24 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 8989 -j DROP

# Make persistent
sudo apt install iptables-persistent
sudo netfilter-persistent save
```

### macOS (if testing locally)

```bash
# Allow the obsidianoid binary through the app firewall
sudo /usr/libexec/ApplicationFirewall/socketfilterfw --add /usr/local/bin/obsidianoid
sudo /usr/libexec/ApplicationFirewall/socketfilterfw --unblockapp /usr/local/bin/obsidianoid
```

---

## Updating

```bash
cd /opt/obsidianoid
git pull                  # or copy new files
sudo make install         # rebuilds, copies binary + static, restarts service
```

---

## Uninstalling

```bash
sudo make uninstall
# Manually clean up if desired:
rm -rf /opt/obsidianoid
rm -f  ~/.obsidianoid.json
rm -rf ~/.obsidianoid
```
