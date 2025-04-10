cat > api/handlers.go << 'EOF'
package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"raft3d/models"
	"raft3d/store"
)

// Handler handles HTTP requests
type Handler struct {
	raftServer *store.RaftServer
}

// NewHandler creates a new API handler
func NewHandler(raftServer *store.RaftServer) *Handler {
	return &Handler{
		raftServer: raftServer,
	}
}

// CreatePrinter handles printer creation
func (h *Handler) CreatePrinter(c *gin.Context) {
	var printer models.Printer
	if err := c.ShouldBindJSON(&printer); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create command for Raft
	data, err := json.Marshal(printer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	cmd := models.Command{
		Type: models.CreatePrinter,
		Data: data,
	}

	// Apply command
	result, err := h.raftServer.ApplyCommand(cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// GetPrinters returns all printers
func (h *Handler) GetPrinters(c *gin.Context) {
	store := h.raftServer.GetStore()
	printers := make([]models.Printer, 0, len(store.Printers))
	
	for _, printer := range store.Printers {
		printers = append(printers, printer)
	}
	
	c.JSON(http.StatusOK, printers)
}

// CreateFilament handles filament creation
func (h *Handler) CreateFilament(c *gin.Context) {
	var filament models.Filament
	if err := c.ShouldBindJSON(&filament); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create command for Raft
	data, err := json.Marshal(filament)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	cmd := models.Command{
		Type: models.CreateFilament,
		Data: data,
	}

	// Apply command
	result, err := h.raftServer.ApplyCommand(cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// GetFilaments returns all filaments
func (h *Handler) GetFilaments(c *gin.Context) {
	store := h.raftServer.GetStore()
	filaments := make([]models.Filament, 0, len(store.Filaments))
	
	for _, filament := range store.Filaments {
		filaments = append(filaments, filament)
	}
	
	c.JSON(http.StatusOK, filaments)
}

// CreatePrintJob handles print job creation
func (h *Handler) CreatePrintJob(c *gin.Context) {
	var printJob models.PrintJob
	if err := c.ShouldBindJSON(&printJob); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create command for Raft
	data, err := json.Marshal(printJob)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	cmd := models.Command{
		Type: models.CreatePrintJob,
		Data: data,
	}

	// Apply command
	result, err := h.raftServer.ApplyCommand(cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// GetPrintJobs returns all print jobs
func (h *Handler) GetPrintJobs(c *gin.Context) {
	store := h.raftServer.GetStore()
	
	// Filter by status if provided
	status := c.Query("status")
	
	printJobs := make([]models.PrintJob, 0)
	for _, job := range store.PrintJobs {
		if status == "" || string(job.Status) == status {
			printJobs = append(printJobs, job)
		}
	}
	
	c.JSON(http.StatusOK, printJobs)
}

// UpdatePrintJobStatus updates a print job status
func (h *Handler) UpdatePrintJobStatus(c *gin.Context) {
	id := c.Param("id")
	status := c.Query("status")
	
	if status == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status query parameter is required"})
		return
	}
	
	cmd := models.Command{
		Type:   models.UpdatePrintJob,
		ID:     id,
		Status: status,
	}
	
	// Apply command
	result, err := h.raftServer.ApplyCommand(cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, result)
}

// JoinCluster handles joining a node to the cluster
func (h *Handler) JoinCluster(c *gin.Context) {
	var req struct {
		NodeID string `json:"node_id"`
		Addr   string `json:"addr"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	if err := h.raftServer.Join(req.NodeID, req.Addr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetClusterInfo returns information about the Raft cluster
func (h *Handler) GetClusterInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"leader":    h.raftServer.GetLeader(),
		"is_leader": h.raftServer.IsLeader(),
	})
}
EOF