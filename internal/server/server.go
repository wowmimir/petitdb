package server

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/wowmimir/petitdb/internal/config"
	"github.com/wowmimir/petitdb/internal/dispatcher"
	"github.com/wowmimir/petitdb/internal/persistence"
	"github.com/wowmimir/petitdb/internal/protocol/resp"
	"github.com/wowmimir/petitdb/internal/pubsub"
	"github.com/wowmimir/petitdb/internal/storage"
)

type Server struct {
	cfg        *config.Config
	listener   net.Listener
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	dispatcher *dispatcher.Dispatcher
	store      *storage.Store
	cleanupWg  sync.WaitGroup
	pm         *persistence.SnapshotManager
	broker     *pubsub.Broker
}

func NewServer(cfg *config.Config, disp *dispatcher.Dispatcher, store *storage.Store, broker *pubsub.Broker) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	pm := persistence.NewSnapshotManager(cfg.Dir)

	return &Server{
		cfg:        cfg,
		ctx:        ctx,
		cancel:     cancel,
		dispatcher: disp,
		store:      store,
		pm:         pm,
		broker:     broker,
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

	// Start cleanup loop
	s.cleanupWg.Add(1)
	go s.cleanupLoop()

	// Accept connections
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

	ticker := time.NewTicker(1 * time.Second)
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

	addr := conn.RemoteAddr()
	log.Printf("Client connected: %s", addr)
	defer log.Printf("Client disconnected: %s", addr)

	// Create the subscriber channel for this client
	// Buffer size: 64 messages (enough to absorb bursts)
	pubsubCh := make(chan []byte, 64)
	defer func() {
		// Clean up: unsubscribe from all topics and close channel
		s.broker.UnsubscribeAll(pubsubCh)
		close(pubsubCh)
	}()

	reader := bufio.NewReader(conn)
	isSubscriber := false

	for {
		select {
		case <-s.ctx.Done():
			log.Printf("Shutting down client %s", addr)
			return
		default:
		}

		// If we're in subscriber mode, we don't read commands anymore
		if isSubscriber {
			s.handleSubscriberMode(conn, pubsubCh, addr)
			return
		}

		// Read command with timeout
		conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		cmd, args, err := resp.ParseCommand(reader)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			log.Printf("Parse error from %s: %v", addr, err)
			conn.Write([]byte(resp.Serialize(err)))
			return
		}

		// Dispatch command with the client's subscriber channel
		result, err := s.dispatcher.Dispatch(cmd, args, pubsubCh)
		if err != nil {
			conn.Write([]byte(resp.Serialize(err)))
			continue
		}

		// Check if this was a SUBSCRIBE command
		if strings.ToUpper(cmd) == "SUBSCRIBE" {
			// Send subscription confirmations
			// result should be []interface{} of confirmations
			if confirmations, ok := result.([]interface{}); ok {
				for _, confirm := range confirmations {
					conn.Write([]byte(resp.Serialize(confirm)))
				}
			}
			// Enter subscriber mode
			isSubscriber = true
			// Continue to subscriber loop (without blocking the main read loop)
			s.handleSubscriberMode(conn, pubsubCh, addr)
			return
		}

		// Normal response for non-SUBSCRIBE commands
		conn.Write([]byte(resp.Serialize(result)))
	}
}

// handleSubscriberMode forwards pub/sub messages to the client.
// This is a separate function to keep the main loop clean.
func (s *Server) handleSubscriberMode(conn net.Conn, pubsubCh chan []byte, addr net.Addr) {
	for {
		select {
		case <-s.ctx.Done():
			log.Printf("Shutting down subscriber %s", addr)
			return
		case msg, ok := <-pubsubCh:
			if !ok {
				// Channel closed
				return
			}
			// Write message to client
			if _, err := conn.Write(msg); err != nil {
				log.Printf("Write error to subscriber %s: %v", addr, err)
				return
			}
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