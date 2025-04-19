package api

import (
	"errors"
	"fmt"
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

// Filament represents a filament roll used for 3D printing
type Filament struct {
	ID                      string `json:"id"`
	Name                    string `json:"name"`
	Type                    string `json:"type"` // PLA, PETG, ABS or TPU
	Color                   string `json:"color"`
	TotalWeightInGrams      int    `json:"total_weight_in_grams"`
	RemainingWeightInGrams  int    `json:"remaining_weight_in_grams"`
}

// PrintJob represents a job to print an item
type PrintJob struct {
	ID                string `json:"id"`
	PrinterID         string `json:"printer_id"`
	FilamentID        string `json:"filament_id"`
	FilePath          string `json:"filepath"`
	PrintWeightInGrams int   `json:"print_weight_in_grams"`
	Status            string `json:"status"` // Queued, Running, Done, Canceled
}

// ValidateFilamentType checks if the provided filament type is valid
func ValidateFilamentType(filamentType string) bool {
	validTypes := []string{"PLA", "PETG", "ABS", "TPU"}
	for _, t := range validTypes {
		if t == filamentType {
			return true
		}
	}
	return false
}

// ValidatePrintJobStatus checks if the status transition is valid
func ValidatePrintJobStatusTransition(currentStatus, newStatus string) error {
	switch currentStatus {
	case "Queued":
		if newStatus == "Running" || newStatus == "Canceled" {
			return nil
		}
	case "Running":
		if newStatus == "Done" || newStatus == "Canceled" {
			return nil
		}
	default:
		return errors.New("invalid status transition: job is already in a terminal state")
	}
	
	return fmt.Errorf("invalid status transition: cannot change from %s to %s", currentStatus, newStatus)
}