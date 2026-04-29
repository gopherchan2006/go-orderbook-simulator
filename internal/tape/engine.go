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

// Config controls how the tape engine generates synthetic trades.
type Config struct {
	// TickInterval is how often we generate a trade tick.
	TickInterval time.Duration
	// BaseQtyMin and BaseQtyMax define the random trade size range (in base asset units).
	BaseQtyMin float64
	BaseQtyMax float64
	// ImbalanceBias scales how much book imbalance shifts aggressor side probability.
	// 0.0 = 50/50 always, 1.0 = fully determined by imbalance.
	ImbalanceBias float64
	// ImbalanceDepth is how many top levels to use for imbalance calculation.
	ImbalanceDepth int
}

// DefaultConfig returns a reasonable default for the sim tape engine.
func DefaultConfig() Config {
	return Config{
		TickInterval:   150 * time.Millisecond,
		BaseQtyMin:     0.01,
		BaseQtyMax:     0.5,
		ImbalanceBias:  0.7,
		ImbalanceDepth: 5,
	}
}

// Engine generates synthetic trade ticks from the order book state.
type Engine struct {
	cfg Config
	ob  *orderbook.OrderBook
	h   *hub.Hub
	seq int64
	rng *rand.Rand
}

// New creates a new tape engine.
func New(cfg Config, ob *orderbook.OrderBook, h *hub.Hub) *Engine {
	return &Engine{
		cfg: cfg,
		ob:  ob,
		h:   h,
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Run starts the tape engine loop. Blocks until ctx is cancelled.
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
	// Determine aggressor side based on imbalance.
	imbalance := e.ob.TopNImbalance(e.cfg.ImbalanceDepth)
	// imbalance in [-1,1]: positive means more bids (buyers dominant → more market buys)
	// P(buy) = 0.5 + bias * imbalance/2
	buyProb := 0.5 + e.cfg.ImbalanceBias*(imbalance/2.0)
	buyProb = clamp(buyProb, 0.05, 0.95)

	side := protocol.Sell
	if e.rng.Float64() < buyProb {
		side = protocol.Buy
	}

	qty := e.cfg.BaseQtyMin + e.rng.Float64()*(e.cfg.BaseQtyMax-e.cfg.BaseQtyMin)
	qty = roundQty(qty)

	// Match against book to get fill price(s). We broadcast each fill as a trade.
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
	// Round to 2 decimal places.
	return float64(int(q*100+0.5)) / 100.0
}
