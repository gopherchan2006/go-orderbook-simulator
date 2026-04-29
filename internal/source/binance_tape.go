package source

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"time"

	"go-orderbook-simulator/internal/hub"
	"go-orderbook-simulator/internal/protocol"

	"github.com/gorilla/websocket"
)

const binanceAggTradeStream = "wss://stream.binance.com:9443/ws/btcusdt@aggTrade"

type binanceAggTradeMsg struct {
	EventType    string `json:"e"`
	EventTime    int64  `json:"E"`
	Symbol       string `json:"s"`
	AggTradeID   int64  `json:"a"`
	Price        string `json:"p"`
	Qty          string `json:"q"`
	IsBuyerMaker bool   `json:"m"`
	BestMatch    bool   `json:"M"`
}

func RunBinanceTape(ctx context.Context, h *hub.Hub) {
	var seq int64
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		err := runBinanceTapeConn(ctx, h, &seq)
		if err != nil && ctx.Err() == nil {
			log.Printf("binance aggTrade: disconnected (%v), reconnecting in 3s...", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
			}
		}
	}
}

func runBinanceTapeConn(ctx context.Context, h *hub.Hub, seq *int64) error {
	log.Printf("binance aggTrade: connecting to %s", binanceAggTradeStream)
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, binanceAggTradeStream, nil)
	if err != nil {
		return err
	}
	defer conn.Close()
	log.Printf("binance aggTrade: connected")

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

		var msg binanceAggTradeMsg
		if err := json.Unmarshal(raw, &msg); err != nil {
			log.Printf("binance aggTrade: parse error: %v", err)
			continue
		}

		price, err := strconv.ParseFloat(msg.Price, 64)
		if err != nil {
			continue
		}
		qty, err := strconv.ParseFloat(msg.Qty, 64)
		if err != nil {
			continue
		}

		side := protocol.Buy
		if msg.IsBuyerMaker {
			side = protocol.Sell
		}

		*seq++
		trade := protocol.Trade{
			Event: "trade",
			Ts:    msg.EventTime,
			Price: price,
			Qty:   qty,
			Side:  side,
			Seq:   *seq,
		}

		data, err := json.Marshal(trade)
		if err != nil {
			log.Printf("binance aggTrade: marshal error: %v", err)
			continue
		}
		h.Broadcast(data)
	}
}
