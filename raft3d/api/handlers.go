package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"fmt"
	
)

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
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/printers")
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

// handleFilaments handles GET and POST requests for filaments
func (s *Server) handleFilaments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetFilaments(w, r)
	case http.MethodPost:
		s.handlePostFilament(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetFilaments handles GET /filaments request
func (s *Server) handleGetFilaments(w http.ResponseWriter, r *http.Request) {
	// Extract filament ID from path if present (for single filament)
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/filaments")
	if path != "" && path != "/" {
		filamentID := strings.TrimPrefix(path, "/")
		s.handleGetFilament(w, filamentID)
		return
	}

	// Get all filaments
	filaments := make(map[string]Filament)
	
	// List all keys with prefix "filament_"
	keys, err := s.store.List("filament_")
	if err != nil {
		http.Error(w, "Failed to retrieve filaments", http.StatusInternalServerError)
		return
	}

	// Get each filament by key
	for _, key := range keys {
		value, err := s.store.Get(key)
		if err != nil {
			continue
		}

		var filament Filament
		if err := json.Unmarshal([]byte(value), &filament); err != nil {
			continue
		}

		filaments[filament.ID] = filament
	}

	// Return the list of filaments
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(filaments)
}

// handleGetFilament handles GET /filaments/{id} request
func (s *Server) handleGetFilament(w http.ResponseWriter, id string) {
	key := "filament_" + id
	value, err := s.store.Get(key)
	if err != nil {
		http.Error(w, "Filament not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(value))
}

// handlePostFilament handles POST /filaments request
func (s *Server) handlePostFilament(w http.ResponseWriter, r *http.Request) {
	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse filament data
	var filament Filament
	if err := json.Unmarshal(body, &filament); err != nil {
		http.Error(w, "Invalid filament data format", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if filament.ID == "" || filament.Name == "" {
		http.Error(w, "Filament ID and Name are required", http.StatusBadRequest)
		return
	}

	// Validate filament type
	if !ValidateFilamentType(filament.Type) {
		http.Error(w, "Invalid filament type. Must be one of: PLA, PETG, ABS, TPU", http.StatusBadRequest)
		return
	}

	// If RemainingWeightInGrams is not set, initialize it to TotalWeightInGrams
	if filament.RemainingWeightInGrams == 0 {
		filament.RemainingWeightInGrams = filament.TotalWeightInGrams
		// Re-serialize to include updated field
		updatedBody, err := json.Marshal(filament)
		if err != nil {
			http.Error(w, "Failed to process filament data", http.StatusInternalServerError)
			return
		}
		body = updatedBody
	}

	// Store filament in the Raft store
	key := "filament_" + filament.ID
	if err := s.store.Set(key, string(body)); err != nil {
		http.Error(w, "Failed to store filament data", http.StatusInternalServerError)
		return
	}

	// Return success
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

// handlePrintJobs handles GET and POST requests for print jobs
func (s *Server) handlePrintJobs(w http.ResponseWriter, r *http.Request) {
	// Check if this is a status update request
	if strings.Contains(r.URL.Path, "/status") && r.Method == http.MethodPost {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) >= 4 {
			jobID := parts[len(parts)-2] // The ID is the second-to-last part
			s.handleUpdatePrintJobStatus(w, r, jobID)
			return
		}
		http.Error(w, "Invalid URL format for status update", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetPrintJobs(w, r)
	case http.MethodPost:
		s.handlePostPrintJob(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetPrintJobs handles GET /print_jobs request
func (s *Server) handleGetPrintJobs(w http.ResponseWriter, r *http.Request) {
	// Extract print job ID from path if present (for single print job)
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/print_jobs")
	if path != "" && path != "/" {
		jobID := strings.TrimPrefix(path, "/")
		s.handleGetPrintJob(w, jobID)
		return
	}

	// Check for status filter query parameter
	statusFilter := r.URL.Query().Get("status")

	// Get all print jobs
	printJobs := make(map[string]PrintJob)
	
	// List all keys with prefix "printjob_"
	keys, err := s.store.List("printjob_")
	if err != nil {
		http.Error(w, "Failed to retrieve print jobs", http.StatusInternalServerError)
		return
	}

	// Get each print job by key
	for _, key := range keys {
		value, err := s.store.Get(key)
		if err != nil {
			continue
		}

		var printJob PrintJob
		if err := json.Unmarshal([]byte(value), &printJob); err != nil {
			continue
		}

		// Apply status filter if specified
		if statusFilter != "" && printJob.Status != statusFilter {
			continue
		}

		printJobs[printJob.ID] = printJob
	}

	// Return the list of print jobs
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(printJobs)
}

// handleGetPrintJob handles GET /print_jobs/{id} request
func (s *Server) handleGetPrintJob(w http.ResponseWriter, id string) {
	key := "printjob_" + id
	value, err := s.store.Get(key)
	if err != nil {
		http.Error(w, "Print job not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(value))
}

// handlePostPrintJob handles POST /print_jobs request
func (s *Server) handlePostPrintJob(w http.ResponseWriter, r *http.Request) {
	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse print job data
	var printJob PrintJob
	if err := json.Unmarshal(body, &printJob); err != nil {
		http.Error(w, "Invalid print job data format", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if printJob.ID == "" || printJob.PrinterID == "" || printJob.FilamentID == "" || printJob.FilePath == "" || printJob.PrintWeightInGrams <= 0 {
		http.Error(w, "All fields are required: ID, PrinterID, FilamentID, FilePath, and PrintWeightInGrams (> 0)", http.StatusBadRequest)
		return
	}

	// Validate printer exists
	printerKey := "printer_" + printJob.PrinterID
	_, err = s.store.Get(printerKey)
	if err != nil {
		http.Error(w, "Printer not found", http.StatusBadRequest)
		return
	}

	// Validate filament exists
	filamentKey := "filament_" + printJob.FilamentID
	filamentValue, err := s.store.Get(filamentKey)
	if err != nil {
		http.Error(w, "Filament not found", http.StatusBadRequest)
		return
	}

	var filament Filament
	if err := json.Unmarshal([]byte(filamentValue), &filament); err != nil {
		http.Error(w, "Failed to parse filament data", http.StatusInternalServerError)
		return
	}

	// Calculate weight already allocated to active print jobs using this filament
	allocatedWeight, err := s.calculateAllocatedFilamentWeight(printJob.FilamentID)
	if err != nil {
		http.Error(w, "Failed to calculate allocated filament weight", http.StatusInternalServerError)
		return
	}

	// Check if there's enough filament remaining
	if filament.RemainingWeightInGrams - allocatedWeight < printJob.PrintWeightInGrams {
		errMsg := fmt.Sprintf("Not enough filament remaining. Available: %d grams, Requested: %d grams",
			filament.RemainingWeightInGrams - allocatedWeight, printJob.PrintWeightInGrams)
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	// Set initial status to Queued
	printJob.Status = "Queued"

	// Re-serialize to include the status field
	updatedBody, err := json.Marshal(printJob)
	if err != nil {
		http.Error(w, "Failed to process print job data", http.StatusInternalServerError)
		return
	}

	// Store print job in the Raft store
	key := "printjob_" + printJob.ID
	if err := s.store.Set(key, string(updatedBody)); err != nil {
		http.Error(w, "Failed to store print job data", http.StatusInternalServerError)
		return
	}

	// Return success
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	w.Write(updatedBody)
}

// handleUpdatePrintJobStatus handles POST /print_jobs/{id}/status request
func (s *Server) handleUpdatePrintJobStatus(w http.ResponseWriter, r *http.Request, jobID string) {
	// Get new status from query parameters
	newStatus := r.URL.Query().Get("status")
	if newStatus == "" {
		http.Error(w, "Status parameter is required", http.StatusBadRequest)
		return
	}

	// Validate status value
	if newStatus != "Running" && newStatus != "Done" && newStatus != "Canceled" {
		http.Error(w, "Status must be one of: Running, Done, Canceled", http.StatusBadRequest)
		return
	}

	// Get existing print job
	jobKey := "printjob_" + jobID
	jobValue, err := s.store.Get(jobKey)
	if err != nil {
		http.Error(w, "Print job not found", http.StatusNotFound)
		return
	}

	var printJob PrintJob
	if err := json.Unmarshal([]byte(jobValue), &printJob); err != nil {
		http.Error(w, "Failed to parse print job data", http.StatusInternalServerError)
		return
	}

	// Validate status transition
	if err := ValidatePrintJobStatusTransition(printJob.Status, newStatus); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update print job status
	oldStatus := printJob.Status
	printJob.Status = newStatus

	// If status changed to "Done", update filament remaining weight
	if newStatus == "Done" {
		// Get filament
		filamentKey := "filament_" + printJob.FilamentID
		filamentValue, err := s.store.Get(filamentKey)
		if err != nil {
			http.Error(w, "Filament not found", http.StatusInternalServerError)
			return
		}

		var filament Filament
		if err := json.Unmarshal([]byte(filamentValue), &filament); err != nil {
			http.Error(w, "Failed to parse filament data", http.StatusInternalServerError)
			return
		}

		// Reduce filament weight
		filament.RemainingWeightInGrams -= printJob.PrintWeightInGrams
		if filament.RemainingWeightInGrams < 0 {
			filament.RemainingWeightInGrams = 0
		}

		// Update filament in store
		updatedFilamentData, err := json.Marshal(filament)
		if err != nil {
			http.Error(w, "Failed to process filament data", http.StatusInternalServerError)
			return
		}

		if err := s.store.Set(filamentKey, string(updatedFilamentData)); err != nil {
			http.Error(w, "Failed to update filament data", http.StatusInternalServerError)
			return
		}
	}

	// Save updated print job
	updatedJobData, err := json.Marshal(printJob)
	if err != nil {
		http.Error(w, "Failed to process print job data", http.StatusInternalServerError)
		return
	}

	if err := s.store.Set(jobKey, string(updatedJobData)); err != nil {
		http.Error(w, "Failed to update print job data", http.StatusInternalServerError)
		return
	}

	// Return success message
	response := map[string]string{
		"message": fmt.Sprintf("Print job status updated from %s to %s", oldStatus, newStatus),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// calculateAllocatedFilamentWeight calculates the total weight allocated to active print jobs for a filament
func (s *Server) calculateAllocatedFilamentWeight(filamentID string) (int, error) {
	allocatedWeight := 0

	// List all print jobs
	keys, err := s.store.List("printjob_")
	if err != nil {
		return 0, err
	}

	// Check each print job
	for _, key := range keys {
		value, err := s.store.Get(key)
		if err != nil {
			continue
		}

		var printJob PrintJob
		if err := json.Unmarshal([]byte(value), &printJob); err != nil {
			continue
		}

		// Only count jobs using this filament and in active states
		if printJob.FilamentID == filamentID && (printJob.Status == "Queued" || printJob.Status == "Running") {
			allocatedWeight += printJob.PrintWeightInGrams
		}
	}

	return allocatedWeight, nil
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