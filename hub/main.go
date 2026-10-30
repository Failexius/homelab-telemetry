package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"telemetry/shared"
	"time"
)

type HubState struct {
	mu    sync.RWMutex
	Nodes map[string]shared.NodeTelemetry `json:"nodes"`
}

// Globaler Speicher für alle Nodes
var state = &HubState{
	Nodes: make(map[string]shared.NodeTelemetry),
}

func main() {
	// Routen registrieren
	http.HandleFunc("/api/telemetry", handleTelemetry)
	http.HandleFunc("/api/state", handleState)

	port := ":8088"
	log.Printf("Sentinel Hub läuft auf Port %s ...", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

// 1. POST /api/telemetry: Nimmt Snapshots vom Agenten entgegen
func handleTelemetry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Nur POST erlaubt", http.StatusMethodNotAllowed)
		return
	}

	var payload shared.NodeTelemetry
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Ungültiges JSON", http.StatusBadRequest)
		return
	}

	// Thread-sicher in die Map schreiben
	state.mu.Lock()
	state.Nodes[payload.NodeID] = payload
	state.mu.Unlock()

	w.WriteHeader(http.StatusOK)
	log.Printf("Telemetry empfangen von: %s (%s)", payload.NodeID, payload.Hostname)
}

// 2. GET /api/state: Liefert den gesamten aktuellen Zustand an Clients/Browser
func handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Nur GET erlaubt", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Thread-sicher lesen (mehrere Leser dürfen gleichzeitig lesen)
	state.mu.RLock()
	defer state.mu.RUnlock()

	json.NewEncoder(w).Encode(state.Nodes)
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
