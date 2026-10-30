# Sentinel Telemetry & Monitoring System

A lightweight, modular telemetry and health-monitoring stack designed for homelabs and distributed nodes (Raspberry Pi, On-Demand Desktop PC, and Mobile Clients).

---

## 🏗️ Architecture

```
[ Nodes: PC / Pi ] --(HTTP / Push)--> [ Central Hub ] <--(API / Polling)--> [ Mobile App ]
     (agent)                              (hub)                                 (mobile)
```

- **`agent/`**: Lightweight Go agent running on the target machine (cross-compiled for Linux x86_64, ARM64 / Raspberry Pi, and Windows). Collects CPU, RAM, Disk, Temperature, and Docker container health metrics.
- **`hub/`**: Aggregator service that receives node telemetry via `/api/telemetry` and provides current state via `/api/state`.
- **`shared/`**: Common Go data structures (`types.go`) shared between the agent, hub, and backend.
- **`mobile/`**: Client dashboard application for iOS & Android.

---

## 🚀 Building & Running

### 1. Build the Agent
```bash
cd agent
# For Local PC:
go build -o agent main.go

# For Raspberry Pi (ARM64):
GOOS=linux GOARCH=arm64 go build -o agent-pi main.go
```

### 2. Run the Hub
```bash
cd hub
go run main.go
# Server listens on :8088 by default
```

---

## 🔒 Security & Privacy
- Zero hardcoded passwords, tokens, or external IP addresses.
- All telemetry communicates over local network or secure VPN mesh (e.g., Tailscale).
