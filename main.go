package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"

	"go-orderbook-simulator/internal/hub"
	"go-orderbook-simulator/internal/orderbook"
	"go-orderbook-simulator/internal/protocol"
	"go-orderbook-simulator/internal/source"
	"go-orderbook-simulator/internal/tape"

	"github.com/gorilla/websocket"
)

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

func handleWebSocket(w http.ResponseWriter, r *http.Request, h *hub.Hub, ob *orderbook.OrderBook) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade error: %v", err)
		return
	}
	log.Printf("client connected: %s", r.RemoteAddr)

	send := make(chan []byte, 64)
	h.Register(send)

	defer func() {
		h.Unregister(send)
		conn.Close()
		log.Printf("client disconnected: %s", r.RemoteAddr)
	}()

	// send snapshot first
	bids, asks := ob.GetSnapshot()
	snap := protocol.DepthSnapshot{Event: "depthSnapshot", Bids: bids, Asks: asks}
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

func main() {
	mode := flag.String("mode", "sim", "source mode: sim | binance")
	scenarioPath := flag.String("scenario", "", "path to scenario JSON (required for sim mode)")
	speed := flag.Float64("speed", 1.0, "sim playback speed (1.0=realtime, 0=instant)")
	port := flag.Int("port", 8080, "WebSocket server port")
	allowOrigin := flag.String("allow-origin", "http://localhost:5173", "CORS allowed origin")
	flag.Parse()

	ob := orderbook.New()
	h := hub.New()
	ctx := context.Background()

	switch *mode {
	case "sim":
		if *scenarioPath == "" {
			log.Fatal("--scenario is required in sim mode")
		}
		scenario, err := source.LoadScenario(*scenarioPath)
		if err != nil {
			log.Fatalf("load scenario: %v", err)
		}
		log.Printf("sim mode: loaded %d updates from %q  speed=%.2f", len(scenario.Updates), *scenarioPath, *speed)
		ob.ApplyUpdate(scenario.Initial.Bids, scenario.Initial.Asks)
		go source.RunSim(ctx, scenario, ob, h, *speed)

	case "binance":
		log.Printf("binance mode: connecting to Binance depth stream...")
		go source.RunBinance(ctx, ob, h)

	default:
		log.Fatalf("unknown mode %q: use sim or binance", *mode)
	}

	// Tape engine: runs in both modes, generates trade ticks from book state.
	tapeCfg := tape.DefaultConfig()
	tapeEngine := tape.New(tapeCfg, ob, h)
	go tapeEngine.Run(ctx)

	http.HandleFunc("/ws", corsMiddleware(*allowOrigin, func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(w, r, h, ob)
	}))

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("listening on ws://localhost%s/ws  mode=%s  allow-origin=%s", addr, *mode, *allowOrigin)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
