package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"raft3d/api"
	"raft3d/raft"
)

func main() {
	var (
		nodeID    = flag.String("id", "", "Node ID")
		httpAddr  = flag.String("http", "127.0.0.1:8000", "HTTP server address")
		raftAddr  = flag.String("raft", "127.0.0.1:9000", "Raft server address")
		joinAddr  = flag.String("join", "", "Address of node to join")
		dataDir   = flag.String("data", "data", "Directory for data storage")
		bootstrap = flag.Bool("bootstrap", false, "Bootstrap the cluster")
	)
	flag.Parse()

	if *nodeID == "" {
		log.Fatal("Node ID is required")
	}

	// Ensure data directory exists
	nodeDataDir := filepath.Join(*dataDir, *nodeID)
	if err := os.MkdirAll(nodeDataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %s", err)
	}

	// Initialize the Raft store
	raftStore, err := raft.NewRaftStore(*nodeID, *raftAddr, nodeDataDir, *bootstrap)
	if err != nil {
		log.Fatalf("Failed to create Raft store: %s", err)
	}

	// Start the HTTP server
	httpServer := api.NewServer(*httpAddr, raftStore)
	if err := httpServer.Start(); err != nil {
		log.Fatalf("Failed to start HTTP server: %s", err)
	}

	// If join address is specified, join the cluster
	if *joinAddr != "" {
		// Wait a bit for the server to initialize
		time.Sleep(1 * time.Second)
		if err := httpServer.JoinCluster(*joinAddr, *nodeID, *raftAddr); err != nil {
			log.Fatalf("Failed to join cluster: %s", err)
		}
	}

	fmt.Printf("KV store started, HTTP: %s, Raft: %s\n", *httpAddr, *raftAddr)

	// Wait for signal to exit
	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt)
	<-terminate
	fmt.Println("KV store shutting down")

	// Shutdown procedures
	if err := httpServer.Stop(); err != nil {
		log.Printf("Error stopping HTTP server: %s", err)
	}
	if err := raftStore.Close(); err != nil {
		log.Printf("Error closing Raft store: %s", err)
	}
}