package server

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/wowmimir/petitdb/internal/config"
	"github.com/wowmimir/petitdb/internal/dispatcher"
	"github.com/wowmimir/petitdb/internal/persistence"
	"github.com/wowmimir/petitdb/internal/protocol/resp"
	"github.com/wowmimir/petitdb/internal/storage"
)

type Server struct {
	cfg *config.Config

	listener net.Listener

	wg sync.WaitGroup

	ctx context.Context

	cancel context.CancelFunc

	dispatcher *dispatcher.Dispatcher

	store *storage.Store

	cleanupWg sync.WaitGroup
	
	pm *persistence.SnapshotManager
}

func NewServer(cfg *config.Config, disp *dispatcher.Dispatcher, store *storage.Store) *Server {

	ctx, cancel := context.WithCancel(context.Background())

	pm := persistence.NewSnapshotManager(cfg.Dir)

	loadedStore, wasLoaded, err := pm.Load()

	if err != nil {
		// This should not happen because Load handles corruption and never returns error
		// But just in case, log and continue
		log.Printf("Warning: unexpected error loading snapshot: %v", err)
	}

	// If we loaded a store from snapshot, we should use it instead of the empty one
	if wasLoaded {
		store = loadedStore
	}

	return &Server{
		cfg:        cfg,
		ctx:        ctx,
		cancel:     cancel,
		dispatcher: disp,
		store:      store,
		pm:         pm, // NEW: store persistence manager
	}
}

func (s *Server) Start() error {

	addr := fmt.Sprintf("%s:%d", s.cfg.Bind, s.cfg.Port)

	listener, err := net.Listen("tcp", addr)

	if err != nil {
		return fmt.Errorf("failed to bind to %s: %w", addr, err)
	}

	s.listener = listener

	log.Printf("PetitDB listening on %s", addr)
	log.Printf("Data directory: %s", s.cfg.Dir)

	s.cleanupWg.Add(1)
	go s.cleanupLoop()

	for {
		conn, err := s.listener.Accept()

		if err != nil {
			select {
			case <-s.ctx.Done():
				log.Println("Stopping accept loop (shutdown)")
				return nil

			default:
				log.Printf("Accept error: %v", err)
				continue
			}
		}

		s.wg.Add(1)
		go s.handleConn(conn)
	}
}

func (s *Server) cleanupLoop() {
	defer s.cleanupWg.Done()

	ticker := time.NewTicker(1 * time.Second) // Cleanup every second
	defer ticker.Stop()

	log.Println("Expiration cleanup started (interval: 1s)")

	for {
		select {
		case <-s.ctx.Done():
			log.Println("Expiration cleanup stopping")
			return
		case <-ticker.C:
			deleted := s.store.DeleteExpired()
			if deleted > 0 {
				log.Printf("Cleanup: removed %d expired keys", deleted)
			}
		}
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	log.Printf("Client connected: %s", conn.RemoteAddr())

	defer log.Printf("Client disconnected: %s", conn.RemoteAddr()) // <-- new
	
	reader := bufio.NewReader(conn)

	for {
		select {
		case <-s.ctx.Done():
			log.Printf("Shutting down client %s", conn.RemoteAddr())
			return
		default:
		}

		conn.SetReadDeadline(time.Now().Add(1 * time.Second))

		// Parse command from the stream
		cmd, args, err := resp.ParseCommand(reader)
		if err != nil {
			// If it's a timeout error, continue
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			// Real error: send error back and close
			log.Printf("Parse error from %s: %v", conn.RemoteAddr(), err)
			conn.Write([]byte(resp.Serialize(err)))
			return
		}

		// Dispatch the command
		result, err := s.dispatcher.Dispatch(cmd, args)
		if err != nil {
			// Send error response
			conn.Write([]byte(resp.Serialize(err)))
		} else {
			// Send success response
			conn.Write([]byte(resp.Serialize(result)))
		}
	}
}

func (s *Server) Shutdown() {

	log.Println("Shutting down server...")

	s.cancel()

	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			log.Printf("Error closing listener: %v", err)
		}
	}

	s.wg.Wait()

	log.Println("Server stopped gracefully")
}
