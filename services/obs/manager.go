package obs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/andreykaipov/goobs"
	"github.com/gorilla/websocket"
)

var ErrUnavailable = errors.New("obs unavailable")

type Manager struct {
	addr     string
	password string
	logger   *slog.Logger

	mu          sync.RWMutex
	client      *goobs.Client
	reconnectCh chan struct{}
}

func NewManager(addr, password string, logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}

	return &Manager{
		addr:        addr,
		password:    password,
		logger:      logger,
		reconnectCh: make(chan struct{}, 1),
	}
}

func (m *Manager) Start(ctx context.Context, retryInterval time.Duration) {
	if err := m.connect(); err != nil {
		m.logger.Error("connection failed; continuing without obs", slog.Any("error", err))
	} else {
		m.logger.Info("connected")
	}

	go m.retryLoop(ctx, retryInterval)
}

func (m *Manager) SwitchScene(scene string) error {
	client := m.Client()
	if client == nil {
		m.requestReconnect()
		return ErrUnavailable
	}

	err := SwitchScene(client, scene)
	if err == nil {
		return nil
	}

	if m.shouldReconnect(err) {
		m.invalidateClient(client, err)
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}

	return err
}

func (m *Manager) Client() *goobs.Client {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.client
}

func (m *Manager) Disconnect() {
	m.mu.Lock()
	client := m.client
	m.client = nil
	m.mu.Unlock()

	if client != nil {
		client.Disconnect()
	}
}

func (m *Manager) retryLoop(ctx context.Context, retryInterval time.Duration) {
	ticker := time.NewTicker(retryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-m.reconnectCh:
		}

		if ctx.Err() != nil {
			return
		}

		if m.Client() != nil {
			continue
		}

		if err := m.connect(); err != nil {
			m.logger.Warn("reconnect failed", slog.Any("error", err))
			if m.Client() != nil {
				continue
			}
			continue
		}

		m.logger.Info("connected")
	}
}

func (m *Manager) requestReconnect() {
	select {
	case m.reconnectCh <- struct{}{}:
	default:
	}
}

func (m *Manager) connect() error {
	client, err := goobs.New(m.addr, goobs.WithPassword(m.password))
	if err != nil {
		return err
	}

	m.mu.Lock()
	previous := m.client
	m.client = client
	m.mu.Unlock()

	if previous != nil {
		previous.Disconnect()
	}

	return nil
}

func (m *Manager) invalidateClient(client *goobs.Client, err error) {
	m.mu.Lock()
	if m.client != client {
		m.mu.Unlock()
		return
	}

	m.client = nil
	m.mu.Unlock()

	m.logger.Warn("connection lost; reconnect scheduled", slog.Any("error", err))
	client.Disconnect()
	m.requestReconnect()
}

func (m *Manager) shouldReconnect(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, io.EOF) ||
		errors.Is(err, net.ErrClosed) ||
		errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, syscall.ECONNRESET) {
		return true
	}

	var closeErr *websocket.CloseError
	if errors.As(err, &closeErr) {
		return true
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "client already disconnected") ||
		strings.Contains(msg, "connection reset by peer") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "use of closed network connection")
}
