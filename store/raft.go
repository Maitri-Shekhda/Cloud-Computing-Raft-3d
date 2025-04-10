cat > store/raft.go << 'EOF'
package store

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"raft3d/models"
)

// RaftServer wraps the Raft functionality
type RaftServer struct {
	raft *raft.Raft
	fsm  *FSM
}

// NewRaftServer creates a new Raft server
func NewRaftServer(nodeID, raftAddr string, dir string, bootstrap bool) (*RaftServer, error) {
	// Create and configure the FSM
	fsm := NewFSM()

	// Set up Raft configuration
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(nodeID)
	config.SnapshotInterval = 30 * time.Second
	config.SnapshotThreshold = 100

	// Set up transport
	addr, err := net.ResolveTCPAddr("tcp", raftAddr)
	if err != nil {
		return nil, err
	}
	transport, err := raft.NewTCPTransport(raftAddr, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return nil, err
	}

	// Create the snapshot store
	snapshotStore, err := raft.NewFileSnapshotStore(dir, 3, os.Stderr)
	if err != nil {
		return nil, err
	}

	// Create the log store and stable store
	logStore, err := raftboltdb.NewBoltStore(filepath.Join(dir, "raft-log.bolt"))
	if err != nil {
		return nil, err
	}
	stableStore, err := raftboltdb.NewBoltStore(filepath.Join(dir, "raft-stable.bolt"))
	if err != nil {
		return nil, err
	}

	// Create the Raft instance
	r, err := raft.NewRaft(config, fsm, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		return nil, err
	}

	// Bootstrap if needed
	if bootstrap {
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      raft.ServerID(nodeID),
					Address: raft.ServerAddress(raftAddr),
				},
			},
		}
		r.BootstrapCluster(configuration)
	}

	return &RaftServer{
		raft: r,
		fsm:  fsm,
	}, nil
}

// GetStore returns the current store state
func (rs *RaftServer) GetStore() *models.Store {
	return rs.fsm.GetStore()
}

// Join joins a node to the Raft cluster
func (rs *RaftServer) Join(nodeID, addr string) error {
	configFuture := rs.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		return err
	}

	// Check if the node already exists
	for _, srv := range configFuture.Configuration().Servers {
		if srv.ID == raft.ServerID(nodeID) || srv.Address == raft.ServerAddress(addr) {
			return nil
		}
	}

	// Add the node
	addFuture := rs.raft.AddVoter(raft.ServerID(nodeID), raft.ServerAddress(addr), 0, 0)
	if err := addFuture.Error(); err != nil {
		return err
	}

	return nil
}

// Leave removes a node from the Raft cluster
func (rs *RaftServer) Leave(nodeID string) error {
	removeFuture := rs.raft.RemoveServer(raft.ServerID(nodeID), 0, 0)
	if err := removeFuture.Error(); err != nil {
		return err
	}
	return nil
}

// IsLeader returns whether this node is the current leader
func (rs *RaftServer) IsLeader() bool {
	return rs.raft.State() == raft.Leader
}

// GetLeader returns the current leader's address
func (rs *RaftServer) GetLeader() string {
	return string(rs.raft.Leader())
}

// ApplyCommand applies a command to the FSM
func (rs *RaftServer) ApplyCommand(cmd models.Command) (interface{}, error) {
	if !rs.IsLeader() {
		return nil, fmt.Errorf("not the leader")
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		return nil, err
	}

	// Apply the command
	future := rs.raft.Apply(data, 5*time.Second)
	if err := future.Error(); err != nil {
		return nil, err
	}

	// Check for errors in the result
	if err, ok := future.Response().(error); ok {
		return nil, err
	}

	return future.Response(), nil
}
EOF