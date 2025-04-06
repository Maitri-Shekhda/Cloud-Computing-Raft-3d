package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid" // For generating UUIDs (string IDs)
	"github.com/hashicorp/raft"
)

// Printer represents a 3D printer.
type Printer struct {
	ID      string `json:"id"`      // Unique ID (string)
	Company string `json:"company"` // Company (e.g., Creality, Prusa)
	Model   string `json:"model"`   // Model (e.g., Ender 3, i3 MK3S+)
	// Other printer attributes...
}

// DataStore represents our data store.
type DataStore struct {
	printers map[string]Printer // Keyed by string ID
	mu       sync.RWMutex
}

// NewDataStore creates a new data store.
func NewDataStore() *DataStore {
	return &DataStore{
		printers: make(map[string]Printer),
	}
}

// Handlers is where we'll define all of our request handlers and assign to a struct
type Handlers struct {
	raft  *raft.Raft // Potentially used, needs to be initialized with raft instance
	store *DataStore
}

func NewHandlers(raftNode *raft.Raft, store *DataStore) *Handlers {
	return &Handlers{
		raft:  raftNode,
		store: store,
	}
}

// CreatePrinterHandler handles POST requests to create a printer.
func (h *Handlers) CreatePrinterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var printer Printer
	if err := json.NewDecoder(r.Body).Decode(&printer); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Generate a UUID for the printer
	printer.ID = uuid.New().String()

	// Raft integration point:
	// 1. Serialize the printer data (e.g., to JSON)
	// 2. Submit the serialized data to the Raft log using `raft.Apply()`
	// 3. Raft will ensure this data is consistently replicated to all nodes.
	// 4. Once committed, the `apply` function on each node will update the data store.

	// For now, we skip Raft and directly update the data store.  THIS IS NOT CONSISTENT
	// IN A DISTRIBUTED ENVIRONMENT!  This is just for demonstration of the HTTP handler.

	h.store.mu.Lock()
	h.store.printers[printer.ID] = printer
	h.store.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(printer); err != nil {
		log.Printf("Error encoding printer: %v", err)
	}
}

// ListPrintersHandler handles GET requests to list all printers.
func (h *Handlers) ListPrintersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.store.mu.RLock()
	printerList := make([]Printer, 0, len(h.store.printers))
	for _, printer := range h.store.printers {
		printerList = append(printerList, printer)
	}
	h.store.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(printerList); err != nil {
		log.Printf("Error encoding printers: %v", err)
	}
}

func main() {
	// Initialize DataStore
	store := NewDataStore()

	// Initialize Handlers
	handlers := NewHandlers(nil, store) // passing nil for now

	// Register Handlers
	http.HandleFunc("/api/v1/printers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handlers.CreatePrinterHandler(w, r)
		} else if r.Method == http.MethodGet {
			handlers.ListPrintersHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Start the server
	port := 8080
	log.Printf("Server listening on port %d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}