package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Config struct {
	ScenarioPath string
	Speed        float64
	Port         int
	AllowOrigin  string
}

type Hub struct {
	mu      sync.Mutex
	clients map[chan []byte]struct{}
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[chan []byte]struct{}),
	}
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
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.clients {
		select {
		case ch <- msg:
		default:
		}
	}
}

type depthSnapshotMsg struct {
	Event string       `json:"e"`
	Bids  [][2]float64 `json:"bids"`
	Asks  [][2]float64 `json:"asks"`
}

type depthUpdateMsg struct {
	Event string       `json:"e"`
	Bids  [][2]float64 `json:"b"`
	Asks  [][2]float64 `json:"a"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func corsMiddleware(allowOrigin string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request, hub *Hub, ob *OrderBook) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade error: %v", err)
		return
	}
	log.Printf("client connected: %s", r.RemoteAddr)

	send := make(chan []byte, 64)
	hub.Register(send)

	defer func() {
		hub.Unregister(send)
		conn.Close()
		log.Printf("client disconnected: %s", r.RemoteAddr)
	}()

	bids, asks := ob.GetSnapshot()
	snap := depthSnapshotMsg{Event: "depthSnapshot", Bids: bids, Asks: asks}
	snapData, err := json.Marshal(snap)
	if err != nil {
		log.Printf("snapshot marshal error: %v", err)
		return
	}
	if err := conn.WriteMessage(websocket.TextMessage, snapData); err != nil {
		log.Printf("snapshot send error: %v", err)
		return
	}

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	for {
		select {
		case msg := <-send:
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Printf("write error: %v", err)
				return
			}
		case <-done:
			return
		}
	}
}

func runUpdateLoop(scenario *Scenario, ob *OrderBook, hub *Hub, speed float64) {
	total := len(scenario.Updates)
	for i, update := range scenario.Updates {
		if speed > 0 && update.DelayMs > 0 {
			delay := time.Duration(float64(update.DelayMs)/speed) * time.Millisecond
			time.Sleep(delay)
		}

		ob.ApplyUpdate(update.Bids, update.Asks)

		msg := depthUpdateMsg{
			Event: "depthUpdate",
			Bids:  nonNil(update.Bids),
			Asks:  nonNil(update.Asks),
		}
		data, err := json.Marshal(msg)
		if err != nil {
			log.Printf("update %d marshal error: %v", i+1, err)
			continue
		}

		hub.Broadcast(data)
		log.Printf("[%d/%d] broadcast depthUpdate bids=%v asks=%v", i+1, total, update.Bids, update.Asks)
	}

	log.Println("scenario complete — server stays active for new connections (will send final snapshot)")
}

func main() {
	var cfg Config
	flag.StringVar(&cfg.ScenarioPath, "scenario", "", "path to scenario JSON file (required)")
	flag.Float64Var(&cfg.Speed, "speed", 1.0, "playback speed multiplier: 1.0=realtime, 0=instant, 0.5=half-speed")
	flag.IntVar(&cfg.Port, "port", 8080, "WebSocket server port")
	flag.StringVar(&cfg.AllowOrigin, "allow-origin", "http://localhost:5173", "CORS allowed origin for the SPA dev-server")
	flag.Parse()

	if cfg.ScenarioPath == "" {
		log.Fatal("--scenario flag is required")
	}

	scenario, err := LoadScenario(cfg.ScenarioPath)
	if err != nil {
		log.Fatalf("load scenario: %v", err)
	}
	log.Printf("loaded scenario: %d updates from %q", len(scenario.Updates), cfg.ScenarioPath)

	ob := NewOrderBook()
	ob.ApplyUpdate(scenario.Initial.Bids, scenario.Initial.Asks)

	hub := NewHub()

	http.HandleFunc("/ws", corsMiddleware(cfg.AllowOrigin, func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(w, r, hub, ob)
	}))

	go runUpdateLoop(scenario, ob, hub, cfg.Speed)

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("listening on ws://localhost%s/ws  speed=%.2f  allow-origin=%s", addr, cfg.Speed, cfg.AllowOrigin)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
