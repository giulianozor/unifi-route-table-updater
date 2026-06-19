# urt — UniFi Route Tracker

A lightweight Go daemon that resolves a DNS name to its public IPv4 address and, if it differs from the destination currently configured on a UniFi static route, updates the route automatically.

## How it works

1. Resolves a DNS name (e.g. `home.example.com`) to an IPv4 address.
2. Fetches a static route from the UniFi controller by its route ID.
3. Compares the resolved IP (with CIDR suffix, default `/32`) against the route's current `destination`.
4. If they differ, updates the route to the new destination.

The check repeats on a configurable interval (default 5 minutes).

When the route is updated, an optional Telegram notification can be sent (disabled by default).

## Usage

```bash
# Build
go build -o urt .

# Run
cp config.example.yaml config.yaml
# edit config.yaml with your details
./urt

# Use alternate config path
./urt -config /etc/urt/config.yaml

# Dry-run (log what would change, make no changes)
./urt -dry-run

# Run once and exit
./urt -once

# List all static routes with their ID, name, and destination
./urt -list-routes
```

## Configuration

All configuration is in a YAML file. An example is provided at `config.example.yaml`:

```yaml
unifi_base_url: "https://192.168.1.1:8443"
unifi_api_key: "your-api-key-here"
unifi_site: "default"
dns_name: "home.example.com"
route_id: "<route-id-from-unifi>"
route_cidr: "/32"
check_interval: "5m"
insecure: true
ca_cert: "/path/to/ca.pem"
verbose: false
# telegram_enabled: false
# telegram_bot_token: "your-bot-token"
# telegram_chat_id: "your-chat-id"
```

| Field | Required | Default | Description |
|---|---|---|---|
| `unifi_base_url` | Yes | — | UniFi Network Controller base URL |
| `unifi_api_key` | Yes | — | UniFi API key |
| `dns_name` | Yes | — | DNS name to resolve |
| `route_id` | Yes | — | ID of the static route to manage (from UniFi, `_id`) |
| `unifi_site` | No | `default` | UniFi site name |
| `route_cidr` | No | `/32` | CIDR suffix appended to the resolved IP for the route destination |
| `check_interval` | No | `5m` | How often to check (Go duration format, e.g. `1m`, `30s`, `1h`) |
| `insecure` | No | `false` | Skip TLS certificate verification (for self-signed UniFi certs) |
| `ca_cert` | No | — | Path to a PEM CA or intermediate certificate to trust for TLS |
| `verbose` | No | `false` | Enable verbose debug output (logs HTTP requests, response bodies on errors) |
| `telegram_enabled` | No | `false` | Send a Telegram notification when the route is updated |
| `telegram_bot_token` | No | — | Telegram bot token (required if `telegram_enabled` is `true`) |
| `telegram_chat_id` | No | — | Telegram chat ID to send the message to (required if `telegram_enabled` is `true`) |

## CLI flags

| Flag | Default | Description |
|---|---|---|
| `-config` | `config.yaml` | Path to YAML config file |
| `-dry-run` | `false` | Log what would change without updating the route |
| `-once` | `false` | Run once and exit |
| `-list-routes` | `false` | List all static routes and exit |
| `-insecure` | `false` | Skip TLS certificate verification |
| `-verbose` | `false` | Enable verbose debug output |
| `-info` | `false` | Query the UniFi integration API for controller info and exit |
| `-list-sites` | `false` | List all sites and exit |
| `-get-route` | `false` | Get the current route configuration as JSON and exit |

## Makefile

| Target | Description |
|---|---|
| `build` | Compile the binary |
| `test` | Run all tests |
| `test-race` | Run all tests with the race detector |
| `clean` | Remove built binary and cache |

## Getting the API key

1. Log into the UniFi controller web UI as an admin.
2. Go to **Settings > System > Advanced** (or **System Settings > Integrations** on newer versions).
3. Under **API**, click **Create API Key**.
4. Give it a name (e.g., `urt`) and choose the appropriate role (read/write access is needed to update routes).
5. Copy the generated key — it will not be shown again.

## Finding the route ID

1. Log into the UniFi controller web UI.
2. Go to **Settings > Routing & Firewall > Static Routes**.
3. Create the route if it doesn't exist (destination can be a placeholder).
4. Use the API key to list static routes and find the `_id`:

```bash
curl -sk -H "X-API-KEY: $API_KEY" \
  https://$CONTROLLER:8443/proxy/network/api/s/default/rest/routing | jq '.'
```
