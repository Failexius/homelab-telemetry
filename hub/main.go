package main

import (
	"encoding/json"
	"log"
	"maps"
	"net/http"
	"sync"
	"telemetry/shared"
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
	stateNodesCopy := maps.Clone(state.Nodes)
	state.mu.RUnlock()

	if err := json.NewEncoder(w).Encode(stateNodesCopy); err != nil {
		log.Printf("State-Encoding fehlgeschlagen: %v", err)
		return
	}
}
