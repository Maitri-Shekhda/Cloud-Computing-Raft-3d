cat > utils/utils.go << 'EOF'
package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// EnsureDir ensures a directory exists
func EnsureDir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", path, err)
		}
	}
	return nil
}

// CleanDir removes all files in a directory but keeps the directory itself
func CleanDir(path string) error {
	dir, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	
	for _, d := range dir {
		os.RemoveAll(filepath.Join(path, d.Name()))
	}
	
	return nil
}

// NodeDataDir returns the data directory for a specific node
func NodeDataDir(baseDir, nodeID string) string {
	return filepath.Join(baseDir, "node-"+nodeID)
}

// PrintLeaderStatus prints a message indicating whether this node is the leader
func PrintLeaderStatus(nodeID string, isLeader bool) {
	if isLeader {
		fmt.Printf("Node %s is the current leader\n", nodeID)
	} else {
		fmt.Printf("Node %s is a follower\n", nodeID)
	}
}
EOF