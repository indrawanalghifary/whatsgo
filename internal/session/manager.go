package session

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"

	_ "github.com/mattn/go-sqlite3"
)

type SessionStatus string

const (
	StatusDisconnected SessionStatus = "disconnected"
	StatusConnecting   SessionStatus = "connecting"
	StatusConnected    SessionStatus = "connected"
	StatusLoggedOut    SessionStatus = "logged_out"
)

type SessionInfo struct {
	ID        string        `json:"id"`
	JID       string        `json:"jid,omitempty"`
	Status    SessionStatus `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

type WhatsAppSession struct {
	ID       string
	Client   *whatsmeow.Client
	Device   *store.Device
	Status   SessionStatus
	QRChan   chan string
	EventCh  chan interface{}
	mu       sync.RWMutex
}

type Manager struct {
	sessions map[string]*WhatsAppSession
	store    *sqlstore.Container
	db       *sql.DB
	mu       sync.RWMutex
}

func NewManager(dbPath string) (*Manager, error) {
	// Initialize database with foreign keys enabled
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Initialize whatsmeow store
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	container, err := sqlstore.New(context.Background(), "sqlite3", dbPath+"?_foreign_keys=on", dbLog)
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	manager := &Manager{
		sessions: make(map[string]*WhatsAppSession),
		store:    container,
		db:       db,
	}

	return manager, nil
}

func (m *Manager) CreateSession(sessionID string) (*WhatsAppSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if session already exists
	if _, exists := m.sessions[sessionID]; exists {
		return nil, fmt.Errorf("session %s already exists", sessionID)
	}

	// Get or create device
	device, err := m.store.GetFirstDevice(context.Background())
	if err != nil {
		// Create new device if none exists
		device = m.store.NewDevice()
	}

	// Create new WhatsApp client
	clientLog := waLog.Stdout("Client-"+sessionID, "DEBUG", true)
	client := whatsmeow.NewClient(device, clientLog)

	session := &WhatsAppSession{
		ID:      sessionID,
		Client:  client,
		Device:  device,
		Status:  StatusDisconnected,
		QRChan:  make(chan string, 1),
		EventCh: make(chan interface{}, 100),
	}

	// Add event handler
	client.AddEventHandler(session.eventHandler)

	m.sessions[sessionID] = session
	return session, nil
}

func (m *Manager) GetSession(sessionID string) (*WhatsAppSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	return session, nil
}

func (m *Manager) ListSessions() []SessionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]SessionInfo, 0, len(m.sessions))
	for id, session := range m.sessions {
		session.mu.RLock()
		info := SessionInfo{
			ID:        id,
			Status:    session.Status,
			CreatedAt: time.Now(), // You might want to store this properly
			UpdatedAt: time.Now(),
		}
		if session.Client.Store.ID != nil {
			info.JID = session.Client.Store.ID.String()
		}
		session.mu.RUnlock()
		sessions = append(sessions, info)
	}

	return sessions
}

func (m *Manager) DeleteSession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	// Disconnect client
	if session.Client.IsConnected() {
		session.Client.Disconnect()
	}

	// Remove from map
	delete(m.sessions, sessionID)

	return nil
}

func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Disconnect all sessions
	for _, session := range m.sessions {
		if session.Client.IsConnected() {
			session.Client.Disconnect()
		}
	}

	// Close database
	return m.db.Close()
}

func (s *WhatsAppSession) Connect() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Client.IsConnected() {
		return fmt.Errorf("session is already connected")
	}

	s.Status = StatusConnecting
	return s.Client.Connect()
}

func (s *WhatsAppSession) Disconnect() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Client.IsConnected() {
		s.Client.Disconnect()
	}
	s.Status = StatusDisconnected
}

func (s *WhatsAppSession) IsLoggedIn() bool {
	return s.Client.Store.ID != nil
}

func (s *WhatsAppSession) eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		log.Printf("Received message in session %s: %s", s.ID, v.Message.GetConversation())
	case *events.QR:
		select {
		case s.QRChan <- v.Codes[0]:
		default:
		}
	case *events.Connected:
		s.mu.Lock()
		s.Status = StatusConnected
		s.mu.Unlock()
		log.Printf("Session %s connected", s.ID)
	case *events.Disconnected:
		s.mu.Lock()
		s.Status = StatusDisconnected
		s.mu.Unlock()
		log.Printf("Session %s disconnected", s.ID)
	case *events.LoggedOut:
		s.mu.Lock()
		s.Status = StatusLoggedOut
		s.mu.Unlock()
		log.Printf("Session %s logged out", s.ID)
	}

	// Send event to channel for external handling
	select {
	case s.EventCh <- evt:
	default:
		// Channel is full, skip this event
	}
}

func (s *WhatsAppSession) SendMessage(jid string, message string) error {
	if !s.Client.IsConnected() {
		return fmt.Errorf("session is not connected")
	}

	targetJID, err := types.ParseJID(jid)
	if err != nil {
		return fmt.Errorf("invalid JID: %w", err)
	}

	_, err = s.Client.SendMessage(context.Background(), targetJID, &waE2E.Message{
		Conversation: &message,
	})

	return err
}

func (s *WhatsAppSession) GetStatus() SessionStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Status
}