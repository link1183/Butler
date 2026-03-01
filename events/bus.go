package events

import (
	"log/slog"
	"sync"
)

type Handler func(Event)

type Bus struct {
	mu        sync.RWMutex
	listeners map[string][]Handler
	logger    *slog.Logger
}

func NewBus(logger *slog.Logger) *Bus {
	return &Bus{
		listeners: make(map[string][]Handler),
		logger:    logger,
	}
}

func (b *Bus) Subscribe(eventType string, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.listeners[eventType] = append(b.listeners[eventType], h)
}

func (b *Bus) Publish(e Event) {
	b.mu.RLock()
	handlers := b.listeners[e.Type]
	b.mu.RUnlock()

	if b.logger != nil {
		b.logger.Debug("published event",
			slog.String("event_type", e.Type),
			slog.Int("subscribers", len(handlers)),
		)
	}

	for _, h := range handlers {
		go h(e)
	}
}
