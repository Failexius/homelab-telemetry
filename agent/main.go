package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"telemetry/shared"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("Fehler beim Abrufen des Hostnamens: %v", err)
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Fehler beim Erstellen des Docker-Clients: %v", err)
	}
	defer cli.Close()

	hubURL := os.Getenv("HUB_URL")
	if hubURL == "" {
		hubURL = "http://localhost:8088"
	}

	ticker := time.NewTicker(1500 * time.Millisecond)
	defer ticker.Stop()

	log.Printf("Agent gestartet (Sende an Hub: %s)", hubURL)
	go watchDockerEvents(ctx, cli)
	for {
		select {
		case <-ctx.Done():
			log.Println("Agent wird beendet …")
			return
		case <-ticker.C:
			snapshot := buildSnapshot(ctx, cli, hostname)

			if err := sendTelemetry(ctx, hubURL, snapshot); err != nil {
				log.Printf("Konnte Daten nicht an Hub senden: %v", err)
			}

			data, err := json.MarshalIndent(snapshot, "", "  ")
			if err != nil {
				log.Printf("Fehler beim Marshaling der Daten: %v", err)
				continue
			}
			fmt.Println(string(data))
		}
	}
}

func getCPUTemperature() *float64 {
	data, err := os.ReadFile("/sys/class/thermal/thermal_zone0/temp")
	if err != nil {
		// Datei existiert nicht (z. B. auf Mac oder Windows) -> einfach nil zurückgeben
		return nil
	}

	raw := strings.TrimSpace(string(data))
	val, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil
	}

	tempC := val / 1000.0
	return &tempC
}

func getContainerLogs(ctx context.Context, cli *client.Client, containerID string) (string, error) {
	logs, err := cli.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Tail:       "30",
	})
	if err != nil {
		log.Printf("Fehler beim Abrufen der Container-Logs: %v", err)
		return "", err
	}
	defer logs.Close()
	bytes, err := io.ReadAll(logs)
	if err != nil {
		log.Printf("Fehler beim Lesen der Container-Logs: %v", err)
		return "", err
	}
	return string(bytes), nil
}

func watchDockerEvents(ctx context.Context, cli *client.Client) {
	// 1. Filter: Wir wollen nur Container-Events für "die" und "oom"
	eventFilter := filters.NewArgs()
	eventFilter.Add("type", "container")
	eventFilter.Add("event", "die")
	eventFilter.Add("event", "oom")

	// 2. Einmalig abonnieren -> liefert 2 Channels zurück!
	msgCh, errCh := cli.Events(ctx, events.ListOptions{Filters: eventFilter})

	// 3. Lauschen, bis das Programm beendet wird
	for {
		select {
		case <-ctx.Done():
			// Agent wird beendet (Ctrl+C) -> Goroutine sauber beenden
			return

		case err := <-errCh:
			log.Printf("Docker Event Stream abgebrochen: %v", err)
			return

		case msg := <-msgCh:
			// Hier ist ein Container gestorben!
			exitCode := msg.Actor.Attributes["exitCode"]
			name := msg.Actor.Attributes["name"]
			shortID := msg.Actor.ID
			if len(shortID) > 12 {
				shortID = shortID[:12]
			}

			// ExitCode "0" = gewollt gestoppt. Alles andere = Absturz/Kill!
			if exitCode != "0" {
				log.Printf("🚨 CRASH DETECTED: Container '%s' (%s) gestorben mit Exit-Code %s!", name, shortID, exitCode)

				logs, err := getContainerLogs(ctx, cli, msg.Actor.ID)
				if err != nil {
					log.Printf("Fehler beim Log-Abruf: %v", err)
				} else {
					log.Printf("--- Letzte Logs von %s ---\n%s\n------------------------", name, logs)
				}
			}
		}
	}
}

func buildSnapshot(ctx context.Context, cli *client.Client, hostname string) shared.NodeTelemetry {
	hostStats, err := collectHostMetrics(ctx)
	if err != nil {
		log.Printf("Host-Metriken fehlgeschlagen: %v", err)
	}
	containerStats, err := collectContainerMetrics(ctx, cli)
	if err != nil {
		log.Printf("Docker-Warnung: %v", err)
	}
	nodeData := shared.NodeTelemetry{
		NodeID:     "faisal-node",
		Hostname:   hostname,
		Timestamp:  time.Now(),
		PowerState: shared.PowerOnline,
		Host:       hostStats,
		Containers: containerStats,
	}
	return nodeData
}

func collectHostMetrics(ctx context.Context) (shared.HostMetrics, error) {
	v, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return shared.HostMetrics{}, err
	}

	cpuPercentages, err := cpu.PercentWithContext(ctx, 0, false)
	if err != nil {
		return shared.HostMetrics{}, err
	}
	cpuPercent := 0.0
	if len(cpuPercentages) > 0 {
		cpuPercent = cpuPercentages[0]
	}

	up, err := host.UptimeWithContext(ctx)
	if err != nil {
		return shared.HostMetrics{}, err
	}

	d, err := disk.UsageWithContext(ctx, "/")
	if err != nil {
		return shared.HostMetrics{}, err
	}

	diskInfo := shared.DiskMetric{
		Name:       "Root",
		MountPoint: "/",
		TotalMB:    d.Total / (1024 * 1024),
		UsedMB:     d.Used / (1024 * 1024),
		Percent:    d.UsedPercent,
	}

	metrics := shared.HostMetrics{
		CpuUsage:       cpuPercent,
		RamUsedMB:      v.Used / (1024 * 1024),
		RamTotalMB:     v.Total / (1024 * 1024),
		RamPercent:     v.UsedPercent,
		Uptime:         up,
		DiskMetric:     []shared.DiskMetric{diskInfo},
		CpuTemperature: getCPUTemperature(),
	}

	return metrics, nil
}

func collectContainerMetrics(ctx context.Context, cli *client.Client) ([]shared.ContainerMetrics, error) {
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		// Wenn Docker gerade nicht läuft: leere Liste zurückgeben, nicht abstürzen!
		return []shared.ContainerMetrics{}, nil
	}

	var list []shared.ContainerMetrics
	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}

		id := c.ID
		if len(id) > 12 {
			id = id[:12]
		}

		list = append(list, shared.ContainerMetrics{
			ID:     id,
			Name:   name,
			Image:  c.Image,
			State:  c.State,
			Health: c.Status,
		})
	}

	return list, nil
}

func sendTelemetry(ctx context.Context, hubURL string, data shared.NodeTelemetry) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hubURL+"/api/telemetry", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{Timeout: 2 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
