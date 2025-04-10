cat > models/models.go << 'EOF'
package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"
)

// CommandType defines the type of command to be executed on the FSM
type CommandType string

const (
	CreatePrinter    CommandType = "CREATE_PRINTER"
	CreateFilament   CommandType = "CREATE_FILAMENT"
	CreatePrintJob   CommandType = "CREATE_PRINT_JOB"
	UpdatePrintJob   CommandType = "UPDATE_PRINT_JOB"
)

// Command represents a command to be applied to the FSM
type Command struct {
	Type   CommandType     `json:"type"`
	Data   json.RawMessage `json:"data"`
	ID     string          `json:"id,omitempty"`
	Status string          `json:"status,omitempty"`
}

// Printer represents a 3D printer
type Printer struct {
	ID      string `json:"id"`
	Company string `json:"company"`
	Model   string `json:"model"`
}

// FilamentType represents the type of filament
type FilamentType string

const (
	PLA  FilamentType = "PLA"
	PETG FilamentType = "PETG"
	ABS  FilamentType = "ABS"
	TPU  FilamentType = "TPU"
)

// Filament represents a filament roll
type Filament struct {
	ID                     string       `json:"id"`
	Type                   FilamentType `json:"type"`
	Color                  string       `json:"color"`
	TotalWeightInGrams     int          `json:"total_weight_in_grams"`
	RemainingWeightInGrams int          `json:"remaining_weight_in_grams"`
}

// PrintJobStatus represents the status of a print job
type PrintJobStatus string

const (
	Queued   PrintJobStatus = "Queued"
	Running  PrintJobStatus = "Running"
	Done     PrintJobStatus = "Done"
	Canceled PrintJobStatus = "Canceled"
)

// PrintJob represents a 3D print job
type PrintJob struct {
	ID                 string        `json:"id"`
	PrinterID          string        `json:"printer_id"`
	FilamentID         string        `json:"filament_id"`
	Filepath           string        `json:"filepath"`
	PrintWeightInGrams int           `json:"print_weight_in_grams"`
	Status             PrintJobStatus `json:"status"`
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
}

// Store represents the data store
type Store struct {
	Printers  map[string]Printer  `json:"printers"`
	Filaments map[string]Filament `json:"filaments"`
	PrintJobs map[string]PrintJob `json:"print_jobs"`
	NextID    map[string]int      `json:"next_id"`
}

// NewStore creates a new store
func NewStore() *Store {
	return &Store{
		Printers:  make(map[string]Printer),
		Filaments: make(map[string]Filament),
		PrintJobs: make(map[string]PrintJob),
		NextID: map[string]int{
			"printer":  1,
			"filament": 1,
			"printjob": 1,
		},
	}
}

// GetNextID returns the next ID for the given entity type
func (s *Store) GetNextID(entityType string) string {
	id := s.NextID[entityType]
	s.NextID[entityType]++
	return strconv.Itoa(id)
}

// ValidatePrintJobStatusTransition checks if the status transition is valid
func ValidatePrintJobStatusTransition(currentStatus, newStatus PrintJobStatus) error {
	switch currentStatus {
	case Queued:
		if newStatus != Running && newStatus != Canceled {
			return errors.New("queued jobs can only transition to running or canceled")
		}
	case Running:
		if newStatus != Done && newStatus != Canceled {
			return errors.New("running jobs can only transition to done or canceled")
		}
	case Done, Canceled:
		return errors.New("jobs in done or canceled state cannot transition to any other state")
	default:
		return fmt.Errorf("unknown status: %s", currentStatus)
	}
	return nil
}

// CheckFilamentAvailability checks if there's enough filament for the print job
func (s *Store) CheckFilamentAvailability(filamentID string, printWeight int) error {
	filament, exists := s.Filaments[filamentID]
	if !exists {
		return fmt.Errorf("filament with ID %s does not exist", filamentID)
	}

	// Calculate weight used by queued and running jobs
	weightUsedByOtherJobs := 0
	for _, job := range s.PrintJobs {
		if job.FilamentID == filamentID && (job.Status == Queued || job.Status == Running) {
			weightUsedByOtherJobs += job.PrintWeightInGrams
		}
	}

	remainingWeight := filament.RemainingWeightInGrams - weightUsedByOtherJobs
	if remainingWeight < printWeight {
		return fmt.Errorf("not enough filament: needs %d grams but only %d grams available", 
			printWeight, remainingWeight)
	}

	return nil
}
