package tape

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"time"

	"go-orderbook-simulator/internal/hub"
	"go-orderbook-simulator/internal/orderbook"
	"go-orderbook-simulator/internal/protocol"
)

type Config struct {
	TickInterval   time.Duration
	BaseQtyMin     float64
	BaseQtyMax     float64
	ImbalanceBias  float64
	ImbalanceDepth int
}

func DefaultConfig() Config {
	return Config{
		TickInterval:   150 * time.Millisecond,
		BaseQtyMin:     0.01,
		BaseQtyMax:     0.5,
		ImbalanceBias:  0.7,
		ImbalanceDepth: 5,
	}
}

type Engine struct {
	cfg Config
	ob  *orderbook.OrderBook
	h   *hub.Hub
	seq int64
	rng *rand.Rand
}

func New(cfg Config, ob *orderbook.OrderBook, h *hub.Hub) *Engine {
	return &Engine{
		cfg: cfg,
		ob:  ob,
		h:   h,
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (e *Engine) Run(ctx context.Context) {
	ticker := time.NewTicker(e.cfg.TickInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.tick()
		}
	}
}

func (e *Engine) tick() {
	imbalance := e.ob.TopNImbalance(e.cfg.ImbalanceDepth)
	buyProb := 0.5 + e.cfg.ImbalanceBias*(imbalance/2.0)
	buyProb = clamp(buyProb, 0.05, 0.95)

	side := protocol.Sell
	if e.rng.Float64() < buyProb {
		side = protocol.Buy
	}

	qty := e.cfg.BaseQtyMin + e.rng.Float64()*(e.cfg.BaseQtyMax-e.cfg.BaseQtyMin)
	qty = roundQty(qty)

	var fills [][2]float64
	if side == protocol.Buy {
		fills = e.ob.ConsumeAsks(qty)
	} else {
		fills = e.ob.ConsumeBids(qty)
	}

	if len(fills) == 0 {
		return
	}

	e.seq++
	now := time.Now().UnixMilli()

	for _, fill := range fills {
		price, fillQty := fill[0], fill[1]
		trade := protocol.Trade{
			Event: "trade",
			Ts:    now,
			Price: price,
			Qty:   fillQty,
			Side:  side,
			Seq:   e.seq,
		}
		data, err := json.Marshal(trade)
		if err != nil {
			log.Printf("tape: marshal error: %v", err)
			continue
		}
		e.h.Broadcast(data)
	}
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func roundQty(q float64) float64 {
	return float64(int(q*100+0.5)) / 100.0
}
