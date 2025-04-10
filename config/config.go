cat > config/config.go << 'EOF'
package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

// Config holds the application configuration
type Config struct {
	NodeID      string
	RaftAddr    string
	HTTPAddr    string
	DataDir     string
	Bootstrap   bool
	JoinAddr    string
}

// ParseFlags parses command line flags to populate the configuration
func ParseFlags() *Config {
	cfg := &Config{}
	
	// Define flags
	flag.StringVar(&cfg.NodeID, "id", "", "Node ID (required)")
	flag.StringVar(&cfg.RaftAddr, "raft-addr", "127.0.0.1:7000", "Raft bind address")
	flag.StringVar(&cfg.HTTPAddr, "http-addr", "127.0.0.1:8000", "HTTP API bind address")
	flag.StringVar(&cfg.DataDir, "data-dir", "", "Data directory (required)")
	flag.BoolVar(&cfg.Bootstrap, "bootstrap", false, "Bootstrap the cluster")
	flag.StringVar(&cfg.JoinAddr, "join", "", "Address of the node to join")
	
	// Parse flags
	flag.Parse()
	
	// Validate required flags
	if cfg.NodeID == "" {
		fmt.Println("Node ID is required")
		flag.Usage()
		os.Exit(1)
	}
	
	if cfg.DataDir == "" {
		fmt.Println("Data directory is required")
		flag.Usage()
		os.Exit(1)
	}
	
	// If both bootstrap and join are provided, error out
	if cfg.Bootstrap && cfg.JoinAddr != "" {
		fmt.Println("Cannot both bootstrap and join")
		flag.Usage()
		os.Exit(1)
	}
	
	return cfg
}

// GetNodeIDFromEnv returns the node ID from an environment variable if set
func GetNodeIDFromEnv() (string, bool) {
	nodeID := os.Getenv("RAFT3D_NODE_ID")
	return nodeID, nodeID != ""
}

// GetPortOffset returns the port offset from an environment variable if set
func GetPortOffset() int {
	offsetStr := os.Getenv("RAFT3D_PORT_OFFSET")
	if offsetStr == "" {
		return 0
	}
	
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		return 0
	}
	
	return offset
}
EOF