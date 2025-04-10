package raft

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb/v2"
	
)

// Store provides an interface for operations on the distributed store
type Store interface {
	// Get retrieves a value for the given key
	Get(key string) (string, error)
	
	// Set sets a value for the given key
	Set(key string, value string) error
	
	// Delete removes a key
	Delete(key string) error
	
	// List returns all keys with a given prefix
	List(prefix string) ([]string, error)
	
	// Join adds a node to the cluster
	Join(nodeID string, addr string) error
	
	// Close closes the store
	Close() error
	
	// Leader returns the current leader's address
	Leader() string
	
	// Metrics returns metrics about the Raft cluster
	Metrics() map[string]interface{}
}

// RaftStore implements the Store interface using Hashicorp's Raft
type RaftStore struct {
	raft          *raft.Raft
	fsm           *FSM
	raftConfig    *raft.Config
	raftBoltStore *raftboltdb.BoltStore
	raftTransport *raft.NetworkTransport
	dataDir       string
}

// NewRaftStore creates a new Raft-backed store
func NewRaftStore(nodeID, raftAddr, dataDir string, bootstrap bool) (*RaftStore, error) {
	// Create the FSM
	fsm := NewFSM()

	// Create Raft config
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(nodeID)
	
	// Set some timeouts appropriate for a demo
	config.HeartbeatTimeout = 500 * time.Millisecond
	config.ElectionTimeout = 500 * time.Millisecond
	config.LeaderLeaseTimeout = 400 * time.Millisecond
	config.CommitTimeout = 100 * time.Millisecond

	// Create Raft transport
	addr, err := net.ResolveTCPAddr("tcp", raftAddr)
	if err != nil {
		return nil, err
	}
	transport, err := raft.NewTCPTransport(raftAddr, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return nil, err
	}

	// Create the snapshot store
	snapshotStore, err := raft.NewFileSnapshotStore(dataDir, 3, os.Stderr)
	if err != nil {
		return nil, err
	}

	// Create the log store and stable store
	boltDB, err := raftboltdb.NewBoltStore(filepath.Join(dataDir, "raft.db"))
	if err != nil {
		return nil, err
	}
	
	// Create Raft instance
	r, err := raft.NewRaft(config, fsm, boltDB, boltDB, snapshotStore, transport)
	if err != nil {
		return nil, err
	}

	// Bootstrap the cluster if needed
	if bootstrap {
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      config.LocalID,
					Address: transport.LocalAddr(),
				},
			},
		}
		r.BootstrapCluster(configuration)
	}

	return &RaftStore{
		raft:          r,
		fsm:           fsm,
		raftConfig:    config,
		raftBoltStore: boltDB,
		raftTransport: transport,
		dataDir:       dataDir,
	}, nil
}

// Get retrieves a value for the given key
func (s *RaftStore) Get(key string) (string, error) {
	return s.fsm.Get(key)
}

// Set sets a value for the given key
func (s *RaftStore) Set(key string, value string) error {
	if s.raft.State() != raft.Leader {
		return fmt.Errorf("not leader")
	}

	cmd := &Command{
		Op:    "set",
		Key:   key,
		Value: value,
	}
	
	data, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	future := s.raft.Apply(data, 10*time.Second)
	return future.Error()
}

// Delete removes a key
func (s *RaftStore) Delete(key string) error {
	if s.raft.State() != raft.Leader {
		return fmt.Errorf("not leader")
	}

	cmd := &Command{
		Op:  "delete",
		Key: key,
	}
	
	data, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	future := s.raft.Apply(data, 10*time.Second)
	return future.Error()
}

// List returns all keys with a given prefix
func (s *RaftStore) List(prefix string) ([]string, error) {
	return s.fsm.List(prefix)
}

// Join adds a node to the cluster
func (s *RaftStore) Join(nodeID string, addr string) error {
	if s.raft.State() != raft.Leader {
		return fmt.Errorf("not leader")
	}

	configFuture := s.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		return err
	}

	// Check if the node already exists
	for _, srv := range configFuture.Configuration().Servers {
		if srv.ID == raft.ServerID(nodeID) || srv.Address == raft.ServerAddress(addr) {
			// Node already exists
			return nil
		}
	}

	// Add the node
	future := s.raft.AddVoter(raft.ServerID(nodeID), raft.ServerAddress(addr), 0, 0)
	if err := future.Error(); err != nil {
		return err
	}

	return nil
}

// Close shuts down the Raft instance and closes the BoltDB store
func (s *RaftStore) Close() error {
	future := s.raft.Shutdown()
	if err := future.Error(); err != nil {
		return err
	}

	if s.raftBoltStore != nil {
		if err := s.raftBoltStore.Close(); err != nil {
			return err
		}
	}

	return nil
}

// Leader returns the current leader's address
func (s *RaftStore) Leader() string {
	return string(s.raft.Leader())
}

// Metrics returns metrics about the Raft cluster
func (s *RaftStore) Metrics() map[string]interface{} {
	leaderAddr := s.raft.Leader()
	
	isLeader := false
	if leaderAddr == s.raftTransport.LocalAddr() {
		isLeader = true
	}

	stats := s.raft.Stats()
	
	metrics := map[string]interface{}{
		"node_id":      string(s.raftConfig.LocalID),
		"state":        s.raft.State().String(),
		"is_leader":    isLeader,
		"leader_addr":  string(leaderAddr),
		"last_contact": stats["last_contact"],
		"term":         stats["term"],
		"last_log_index": stats["last_log_index"],
		"last_log_term":  stats["last_log_term"],
		"commit_index":   stats["commit_index"],
		"applied_index":  stats["applied_index"],
		"fsm_pending":    stats["fsm_pending"],
	}

	return metrics
}