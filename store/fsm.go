cat > store/fsm.go << 'EOF'
package store

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/hashicorp/raft"
	"raft3d/models"
)

// FSM implements the raft.FSM interface for the 3D printer management system
type FSM struct {
	mu    sync.RWMutex
	store *models.Store
}

// NewFSM creates a new FSM with an initialized store
func NewFSM() *FSM {
	return &FSM{
		store: models.NewStore(),
	}
}

// Apply applies a Raft log entry to the FSM
func (f *FSM) Apply(log *raft.Log) interface{} {
	f.mu.Lock()
	defer f.mu.Unlock()

	var cmd models.Command
	if err := json.Unmarshal(log.Data, &cmd); err != nil {
		return fmt.Errorf("failed to unmarshal command: %v", err)
	}

	switch cmd.Type {
	case models.CreatePrinter:
		return f.applyCreatePrinter(cmd.Data)
	case models.CreateFilament:
		return f.applyCreateFilament(cmd.Data)
	case models.CreatePrintJob:
		return f.applyCreatePrintJob(cmd.Data)
	case models.UpdatePrintJob:
		return f.applyUpdatePrintJob(cmd.ID, cmd.Status)
	default:
		return fmt.Errorf("unknown command type: %s", cmd.Type)
	}
}

// Snapshot returns a snapshot of the FSM
func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Clone the store for the snapshot
	snapshot := &models.Store{
		Printers:  make(map[string]models.Printer),
		Filaments: make(map[string]models.Filament),
		PrintJobs: make(map[string]models.PrintJob),
		NextID:    make(map[string]int),
	}

	// Copy printers
	for k, v := range f.store.Printers {
		snapshot.Printers[k] = v
	}

	// Copy filaments
	for k, v := range f.store.Filaments {
		snapshot.Filaments[k] = v
	}

	// Copy print jobs
	for k, v := range f.store.PrintJobs {
		snapshot.PrintJobs[k] = v
	}

	// Copy next IDs
	for k, v := range f.store.NextID {
		snapshot.NextID[k] = v
	}

	return &fsmSnapshot{store: snapshot}, nil
}

// Restore restores the FSM from a snapshot
func (f *FSM) Restore(rc io.ReadCloser) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Read the snapshot data
	data, err := io.ReadAll(rc)
	if err != nil {
		return err
	}

	// Unmarshal the snapshot
	var store models.Store
	if err := json.Unmarshal(data, &store); err != nil {
		return err
	}

	// Replace the current store with the restored one
	f.store = &store
	return nil
}

// applyCreatePrinter applies a CreatePrinter command
func (f *FSM) applyCreatePrinter(data []byte) interface{} {
	var printer models.Printer
	if err := json.Unmarshal(data, &printer); err != nil {
		return err
	}

	// Generate ID if not provided
	if printer.ID == "" {
		printer.ID = f.store.GetNextID("printer")
	}

	// Store the printer
	f.store.Printers[printer.ID] = printer
	return printer
}

// applyCreateFilament applies a CreateFilament command
func (f *FSM) applyCreateFilament(data []byte) interface{} {
	var filament models.Filament
	if err := json.Unmarshal(data, &filament); err != nil {
		return err
	}

	// Generate ID if not provided
	if filament.ID == "" {
		filament.ID = f.store.GetNextID("filament")
	}

	// Set remaining weight to total weight initially
	if filament.RemainingWeightInGrams == 0 {
		filament.RemainingWeightInGrams = filament.TotalWeightInGrams
	}

	// Store the filament
	f.store.Filaments[filament.ID] = filament
	return filament
}

// applyCreatePrintJob applies a CreatePrintJob command
func (f *FSM) applyCreatePrintJob(data []byte) interface{} {
	var printJob models.PrintJob
	if err := json.Unmarshal(data, &printJob); err != nil {
		return err
	}

	// Validate printer and filament exist
	if _, exists := f.store.Printers[printJob.PrinterID]; !exists {
		return fmt.Errorf("printer with ID %s does not exist", printJob.PrinterID)
	}
	if _, exists := f.store.Filaments[printJob.FilamentID]; !exists {
		return fmt.Errorf("filament with ID %s does not exist", printJob.FilamentID)
	}

	// Check filament availability
	if err := f.store.CheckFilamentAvailability(printJob.FilamentID, printJob.PrintWeightInGrams); err != nil {
		return err
	}

	// Generate ID if not provided
	if printJob.ID == "" {
		printJob.ID = f.store.GetNextID("printjob")
	}

	// Set status to Queued
	printJob.Status = models.Queued
	printJob.CreatedAt = time.Now()
	printJob.UpdatedAt = time.Now()

	// Store the print job
	f.store.PrintJobs[printJob.ID] = printJob
	return printJob
}

// applyUpdatePrintJob applies an UpdatePrintJob command
func (f *FSM) applyUpdatePrintJob(id string, status string) interface{} {
	// Validate print job exists
	printJob, exists := f.store.PrintJobs[id]
	if !exists {
		return fmt.Errorf("print job with ID %s does not exist", id)
	}

	// Validate status transition
	newStatus := models.PrintJobStatus(status)
	if err := models.ValidatePrintJobStatusTransition(printJob.Status, newStatus); err != nil {
		return err
	}

	// Update status
	printJob.Status = newStatus
	printJob.UpdatedAt = time.Now()

	// If status is Done, reduce filament weight
	if newStatus == models.Done {
		filament := f.store.Filaments[printJob.FilamentID]
		filament.RemainingWeightInGrams -= printJob.PrintWeightInGrams
		if filament.RemainingWeightInGrams < 0 {
			filament.RemainingWeightInGrams = 0
		}
		f.store.Filaments[printJob.FilamentID] = filament
	}

	// Update the print job
	f.store.PrintJobs[id] = printJob
	return printJob
}

// GetStore returns the current store state
func (f *FSM) GetStore() *models.Store {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.store
}

// fsmSnapshot implements the raft.FSMSnapshot interface
type fsmSnapshot struct {
	store *models.Store
}

// Persist persists the FSM snapshot
func (s *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	// Convert store to JSON
	data, err := json.Marshal(s.store)
	if err != nil {
		sink.Cancel()
		return err
	}

	// Write to sink
	if _, err := sink.Write(data); err != nil {
		sink.Cancel()
		return err
	}

	return sink.Close()
}

// Release releases resources held by the snapshot
func (s *fsmSnapshot) Release() {}
EOF