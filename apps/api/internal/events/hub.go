package events

import "sync"

type Event struct {
	Type string
	Data any
}

type Hub struct {
	mu          sync.Mutex
	subscribers map[chan Event]struct{}
}

func NewHub() *Hub {
	return &Hub{subscribers: make(map[chan Event]struct{})}
}

func (h *Hub) Subscribe() (<-chan Event, func()) {
	channel := make(chan Event, 32)
	h.mu.Lock()
	h.subscribers[channel] = struct{}{}
	h.mu.Unlock()

	return channel, func() {
		h.mu.Lock()
		if _, ok := h.subscribers[channel]; ok {
			delete(h.subscribers, channel)
			close(channel)
		}
		h.mu.Unlock()
	}
}

func (h *Hub) Publish(event Event) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for channel := range h.subscribers {
		select {
		case channel <- event:
		default:
			delete(h.subscribers, channel)
			close(channel)
		}
	}
}
