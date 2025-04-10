cat > api/router.go << 'EOF'
package api

import (
	"github.com/gin-gonic/gin"
	"raft3d/store"
)

// SetupRouter sets up the HTTP router
func SetupRouter(raftServer *store.RaftServer) *gin.Engine {
	router := gin.Default()
	
	// Create handler
	handler := NewHandler(raftServer)
	
	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Printer endpoints
		v1.POST("/printers", handler.CreatePrinter)
		v1.GET("/printers", handler.GetPrinters)
		
		// Filament endpoints
		v1.POST("/filaments", handler.CreateFilament)
		v1.GET("/filaments", handler.GetFilaments)
		
		// Print job endpoints
		v1.POST("/print_jobs", handler.CreatePrintJob)
		v1.GET("/print_jobs", handler.GetPrintJobs)
		v1.POST("/print_jobs/:id/status", handler.UpdatePrintJobStatus)
		
		// Cluster management endpoints
		v1.POST("/join", handler.JoinCluster)
		v1.GET("/cluster", handler.GetClusterInfo)
	}
	
	return router
}
EOF