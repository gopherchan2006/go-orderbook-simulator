package source

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"go-orderbook-simulator/internal/hub"
	"go-orderbook-simulator/internal/orderbook"
	"go-orderbook-simulator/internal/protocol"

	"github.com/gorilla/websocket"
)

const binanceDepthStream = "wss://stream.binance.com:9443/ws/btcusdt@depth@100ms"
const binanceDepthREST = "https://api.binance.com/api/v3/depth?symbol=BTCUSDT&limit=1000"

type binanceDepthMsg struct {
	EventType string      `json:"e"`
	EventTime int64       `json:"E"`
	Symbol    string      `json:"s"`
	FirstID   int64       `json:"U"`
	FinalID   int64       `json:"u"`
	Bids      [][2]string `json:"b"`
	Asks      [][2]string `json:"a"`
}

type binanceRESTSnap struct {
	LastUpdateID int64       `json:"lastUpdateId"`
	Bids         [][2]string `json:"bids"`
	Asks         [][2]string `json:"asks"`
}

func parseLevels(raw [][2]string) []protocol.Level {
	out := make([]protocol.Level, 0, len(raw))
	for _, pair := range raw {
		p, err := strconv.ParseFloat(pair[0], 64)
		if err != nil {
			continue
		}
		q, err := strconv.ParseFloat(pair[1], 64)
		if err != nil {
			continue
		}
		out = append(out, protocol.Level{Price: p, Amount: q})
	}
	return out
}

func fetchRESTSnapshot(ctx context.Context) (*binanceRESTSnap, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, binanceDepthREST, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("REST snapshot: HTTP %d", resp.StatusCode)
	}
	var snap binanceRESTSnap
	if err := json.NewDecoder(resp.Body).Decode(&snap); err != nil {
		return nil, err
	}
	return &snap, nil
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
	log.Printf("binance depth: connected, fetching REST snapshot...")

	snap, err := fetchRESTSnapshot(ctx)
	if err != nil {
		return fmt.Errorf("REST snapshot fetch failed: %w", err)
	}
	log.Printf("binance depth: REST snapshot at lastUpdateId=%d", snap.LastUpdateID)

	snapBids := parseLevels(snap.Bids)
	snapAsks := parseLevels(snap.Asks)
	ob.ResetLevels(snapBids, snapAsks)

	snapMsg, err := json.Marshal(protocol.DepthSnapshot{
		Event: "depthSnapshot",
		Bids:  snapBids,
		Asks:  snapAsks,
	})
	if err == nil {
		h.Broadcast(snapMsg)
	}

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

		// Drop updates already included in REST snapshot.
		if msg.FinalID <= snap.LastUpdateID {
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
