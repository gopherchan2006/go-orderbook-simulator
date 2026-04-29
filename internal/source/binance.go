package source

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"go-orderbook-simulator/internal/hub"
	"go-orderbook-simulator/internal/orderbook"
	"go-orderbook-simulator/internal/protocol"

	"github.com/gorilla/websocket"
)

const binanceDepthStream = "wss://stream.binance.com:9443/ws/btcusdt@depth@100ms"

type binanceDepthMsg struct {
	EventType string               `json:"e"`
	EventTime int64                `json:"E"`
	Symbol    string               `json:"s"`
	FirstID   int64                `json:"U"`
	FinalID   int64                `json:"u"`
	Bids      [][2]json.RawMessage `json:"b"`
	Asks      [][2]json.RawMessage `json:"a"`
}

func parseLevels(raw [][2]json.RawMessage) []protocol.Level {
	out := make([]protocol.Level, 0, len(raw))
	for _, pair := range raw {
		var p, q float64
		if err := json.Unmarshal(pair[0], &p); err != nil {
			continue
		}
		if err := json.Unmarshal(pair[1], &q); err != nil {
			continue
		}
		out = append(out, protocol.Level{Price: p, Amount: q})
	}
	return out
}

func RunBinance(ctx context.Context, ob *orderbook.OrderBook, h *hub.Hub) {
	var seq int64
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		err := runBinanceConn(ctx, ob, h, &seq)
		if err != nil && ctx.Err() == nil {
			log.Printf("binance depth: disconnected (%v), reconnecting in 3s...", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
			}
		}
	}
}

func runBinanceConn(ctx context.Context, ob *orderbook.OrderBook, h *hub.Hub, seq *int64) error {
	log.Printf("binance depth: connecting to %s", binanceDepthStream)
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, binanceDepthStream, nil)
	if err != nil {
		return err
	}
	defer conn.Close()
	log.Printf("binance depth: connected")

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		_, raw, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		var msg binanceDepthMsg
		if err := json.Unmarshal(raw, &msg); err != nil {
			log.Printf("binance depth: parse error: %v", err)
			continue
		}

		bids := parseLevels(msg.Bids)
		asks := parseLevels(msg.Asks)

		ob.ApplyLevels(bids, asks)
		*seq++

		update := protocol.DepthUpdate{
			Event: "depthUpdate",
			Seq:   *seq,
			Ts:    msg.EventTime,
			Bids:  bids,
			Asks:  asks,
		}
		data, err := json.Marshal(update)
		if err != nil {
			log.Printf("binance depth: marshal error: %v", err)
			continue
		}
		h.Broadcast(data)
	}
}
