package hub

import "sync"

type Hub struct {
	mu      sync.Mutex
	clients map[chan []byte]struct{}
	hooks   []func([]byte)
}

func New() *Hub {
	return &Hub{clients: make(map[chan []byte]struct{})}
}

func (h *Hub) AddHook(fn func([]byte)) {
	h.hooks = append(h.hooks, fn)
}

func (h *Hub) Register(ch chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[ch] = struct{}{}
}

func (h *Hub) Unregister(ch chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, ch)
}

func (h *Hub) Broadcast(msg []byte) {
	for _, hook := range h.hooks {
		hook(msg)
	}

	h.mu.Lock()
	clients := make([]chan []byte, 0, len(h.clients))
	for ch := range h.clients {
		clients = append(clients, ch)
	}
	h.mu.Unlock()

	for _, ch := range clients {
		select {
		case ch <- msg:
		default:
		}
	}
}
