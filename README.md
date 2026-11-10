# Sentinel Telemetry

Small telemetry and container-monitoring setup for a two-node homelab (a 24/7 Raspberry Pi 3B+ and an on-demand Windows/WSL2 desktop) connected over Tailscale.

Because the Raspberry Pi runs with a strict 1 GB RAM limit alongside existing services, the Go binaries (`agent` and `hub`) are kept intentionally minimal (~10 MB RSS on ARM64).

## Structure

```
[ Nodes: PC / Pi ] --(HTTP POST)--> [ Central Hub ] <--(HTTP)--> [ Client ]
     (agent)                             (hub)                (planned: mobile app)
```

- `agent/`: Daemon running on each host. Collects system vitals (CPU, memory, disk, thermal sensors via `gopsutil`) and watches the local Docker socket for container status and crash/OOM events. Pushes snapshots to the hub every 1.5 seconds.
- `hub/`: Central service running on the Pi (`:8088`). Keeps the latest state of all nodes in memory (`POST /api/telemetry`, `GET /api/state`) and serves as the backend for the client app.
- `shared/`: Shared Go structs (`types.go`) for telemetry payloads, container metrics, and incident events.

### Planned

- **Mobile dashboard:** a Flutter client that reads the hub state. Not part of the repository yet.
- **Live updates:** pushing state changes from the hub over WebSocket instead of polling `GET /api/state`.

## Building and Running

### Agent

Build locally or cross-compile for the target nodes:

```bash
cd agent

# Local build
go build -o agent main.go

# Raspberry Pi 3B+ (Linux ARM64)
GOOS=linux GOARCH=arm64 go build -o agent-pi main.go

# Desktop PC (Windows AMD64)
GOOS=windows GOARCH=amd64 go build -o agent.exe main.go
```

By default, the agent sends data to `http://localhost:8088`. Set `HUB_URL` to point it to the central hub:

```bash
HUB_URL="http://100.85.21.9:8088" ./agent
```

### Hub

```bash
cd hub
go run main.go
```

The hub listens on `:8088` by default.

## Network and Configuration

No credentials or external IPs are hardcoded in the repository. Node-to-hub communication is meant to run over the local network or an encrypted mesh VPN like Tailscale.
