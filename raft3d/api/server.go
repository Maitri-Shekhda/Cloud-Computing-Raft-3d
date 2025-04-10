package api

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"raft3d/raft"
)

// Server represents the API server and its dependencies
type Server struct {
	Addr     string
	store    *raft.RaftStore
	httpSrv  *http.Server
}

// NewServer constructs a new API server instance
func NewServer(addr string, store *raft.RaftStore) *Server {
	return &Server{
		Addr:  addr,
		store: store,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Register all route handlers
	mux.HandleFunc("/printers", s.handlePrinters)
	mux.HandleFunc("/printers/", s.handlePrinters) // for /printers/{id}
	mux.HandleFunc("/join", s.handleJoin)
	mux.HandleFunc("/metrics", s.handleMetrics)

	s.httpSrv = &http.Server{
		Addr:    s.Addr,
		Handler: mux,
	}

	log.Printf("Starting HTTP server at %s\n", s.Addr)
	go func() {
		if err := s.httpSrv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %s", err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the HTTP server
func (s *Server) Stop() error {
	if s.httpSrv != nil {
		log.Println("Shutting down HTTP server")
		return s.httpSrv.Close()
	}
	return nil
}

// JoinCluster joins the current node to an existing cluster
func (s *Server) JoinCluster(joinAddr, nodeID, raftAddr string) error {
	url := fmt.Sprintf("http://%s/join", joinAddr)

	reqBody := fmt.Sprintf(`{"node_id":"%s", "raft_addr":"%s"}`, nodeID, raftAddr)
	resp, err := http.Post(url, "application/json", 
	                      strings.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("failed to send join request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("join request failed: %s", resp.Status)
	}

	return nil
}
