package raft

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/hashicorp/raft"
)

// Command represents an action to be performed on the key-value store
type Command struct {
	Op    string `json:"op"`    // Operation: "set" or "delete"
	Key   string `json:"key"`   // Key
	Value string `json:"value"` // Value (used for "set" operations)
}

// FSM implements the raft.FSM interface for a key-value store
type FSM struct {
	mutex sync.RWMutex
	data  map[string]string
}

// NewFSM creates a new FSM instance
func NewFSM() *FSM {
	return &FSM{
		data: make(map[string]string),
	}
}

// Apply applies a Raft log entry to the FSM
func (f *FSM) Apply(log *raft.Log) interface{} {
	var cmd Command
	if err := json.Unmarshal(log.Data, &cmd); err != nil {
		return fmt.Errorf("failed to unmarshal command: %s", err)
	}

	f.mutex.Lock()
	defer f.mutex.Unlock()

	switch cmd.Op {
	case "set":
		f.data[cmd.Key] = cmd.Value
		return nil
	case "delete":
		delete(f.data, cmd.Key)
		return nil
	default:
		return fmt.Errorf("unknown command operation: %s", cmd.Op)
	}
}

// Snapshot returns a snapshot of the FSM
func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	// Copy the data map to make the snapshot
	data := make(map[string]string)
	for k, v := range f.data {
		data[k] = v
	}

	return &FSMSnapshot{data: data}, nil
}

// Restore restores the FSM to a previous state
func (f *FSM) Restore(closer io.ReadCloser) error {
	defer closer.Close()

	data := make(map[string]string)
	if err := json.NewDecoder(closer).Decode(&data); err != nil {
		return err
	}

	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.data = data
	return nil
}

// Get retrieves a value for the given key
func (f *FSM) Get(key string) (string, error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	value, exists := f.data[key]
	if !exists {
		return "", fmt.Errorf("key not found: %s", key)
	}
	return value, nil
}

// List returns all keys with a given prefix
func (f *FSM) List(prefix string) ([]string, error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	var keys []string
	for k := range f.data {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	return keys, nil
}

// FSMSnapshot is a snapshot of the FSM state
type FSMSnapshot struct {
	data map[string]string
}

// Persist writes the snapshot to the given sink
func (s *FSMSnapshot) Persist(sink raft.SnapshotSink) error {
	err := func() error {
		// Encode data
		if err := json.NewEncoder(sink).Encode(s.data); err != nil {
			return err
		}
		return sink.Close()
	}()

	if err != nil {
		sink.Cancel()
	}

	return err
}

// Release is a no-op
func (s *FSMSnapshot) Release() {}