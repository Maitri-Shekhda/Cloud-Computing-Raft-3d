package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// Printer represents a 3D printer in the system
type Printer struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Model       string `json:"model"`
	Status      string `json:"status"`
	Temperature int    `json:"temperature"`
	Material    string `json:"material"`
}

// handlePrinters handles GET and POST requests for printers
func (s *Server) handlePrinters(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetPrinters(w, r)
	case http.MethodPost:
		s.handlePostPrinter(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetPrinters handles GET /printers request
func (s *Server) handleGetPrinters(w http.ResponseWriter, r *http.Request) {
	// Extract printer ID from path if present (for single printer)
	path := strings.TrimPrefix(r.URL.Path, "/printers")
	if path != "" && path != "/" {
		printerID := strings.TrimPrefix(path, "/")
		s.handleGetPrinter(w, printerID)
		return
	}

	// Get all printers
	printers := make(map[string]Printer)
	
	// List all keys with prefix "printer_"
	keys, err := s.store.List("printer_")
	if err != nil {
		http.Error(w, "Failed to retrieve printers", http.StatusInternalServerError)
		return
	}

	// Get each printer by key
	for _, key := range keys {
		value, err := s.store.Get(key)
		if err != nil {
			continue
		}

		var printer Printer
		if err := json.Unmarshal([]byte(value), &printer); err != nil {
			continue
		}

		printers[printer.ID] = printer
	}

	// Return the list of printers
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(printers)
}

// handleGetPrinter handles GET /printers/{id} request
func (s *Server) handleGetPrinter(w http.ResponseWriter, id string) {
	key := "printer_" + id
	value, err := s.store.Get(key)
	if err != nil {
		http.Error(w, "Printer not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(value))
}

// handlePostPrinter handles POST /printers request
func (s *Server) handlePostPrinter(w http.ResponseWriter, r *http.Request) {
	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse printer data
	var printer Printer
	if err := json.Unmarshal(body, &printer); err != nil {
		http.Error(w, "Invalid printer data format", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if printer.ID == "" || printer.Name == "" {
		http.Error(w, "Printer ID and Name are required", http.StatusBadRequest)
		return
	}

	// Store printer in the Raft store
	key := "printer_" + printer.ID
	if err := s.store.Set(key, string(body)); err != nil {
		http.Error(w, "Failed to store printer data", http.StatusInternalServerError)
		return
	}

	// Return success
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

// handleJoin handles requests to join the cluster
func (s *Server) handleJoin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		NodeID   string `json:"node_id"`
		RaftAddr string `json:"raft_addr"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Failed to parse request", http.StatusBadRequest)
		return
	}

	if req.NodeID == "" || req.RaftAddr == "" {
		http.Error(w, "Node ID and Raft address are required", http.StatusBadRequest)
		return
	}

	if err := s.store.Join(req.NodeID, req.RaftAddr); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleMetrics returns metrics about the cluster
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics := s.store.Metrics()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}