cat > main/main.go << 'EOF'
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"raft3d/api"
	"raft3d/config"
	"raft3d/store"
	"raft3d/utils"
)

func main() {
	// Parse command line flags
	cfg := config.ParseFlags()

	// Create node data directory
	nodeDir := utils.NodeDataDir(cfg.DataDir, cfg.NodeID)
	if err := utils.EnsureDir(nodeDir); err != nil {
		log.Fatalf("Failed to create node directory: %v", err)
	}

	// Initialize Raft server
	raftServer, err := store.NewRaftServer(cfg.NodeID, cfg.RaftAddr, nodeDir, cfg.Bootstrap)
	if err != nil {
		log.Fatalf("Failed to start Raft server: %v", err)
	}

	// Set up HTTP server
	router := api.SetupRouter(raftServer)
	httpServer := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: router,
	}

	// Start HTTP server in a goroutine
	go func() {
		log.Printf("Starting HTTP server on %s", cfg.HTTPAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Join the cluster if specified
	if cfg.JoinAddr != "" {
		joinCluster(cfg.JoinAddr, cfg.NodeID, cfg.RaftAddr)
	}

	// Print initial status
	time.Sleep(1 * time.Second) // Wait for Raft to initialize
	utils.PrintLeaderStatus(cfg.NodeID, raftServer.IsLeader())

	// Handle graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")

	// Close HTTP server
	httpServer.Close()

	log.Println("Shutdown complete")
}

// joinCluster joins a node to an existing cluster
func joinCluster(joinAddr, nodeID, raftAddr string) {
	// Wait for the server to be up
	time.Sleep(1 * time.Second)

	// Prepare join request
	joinURL := fmt.Sprintf("http://%s/api/v1/join", joinAddr)
	body := map[string]string{
		"node_id": nodeID,
		"addr":    raftAddr,
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		log.Fatalf("Failed to marshal join request: %v", err)
	}

	// Send join request
	resp, err := http.Post(joinURL, "application/json", bytes.NewBuffer(bodyJSON))
	if err != nil {
		log.Fatalf("Failed to join cluster: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Failed to join cluster, status code: %d", resp.StatusCode)
	}

	log.Printf("Successfully joined the cluster at %s", joinAddr)
}
EOF