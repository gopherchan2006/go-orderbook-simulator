package source

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"go-orderbook-simulator/internal/hub"
	"go-orderbook-simulator/internal/orderbook"
	"go-orderbook-simulator/internal/protocol"
)

// ScenarioUpdate is one step in the sim scenario.
type ScenarioUpdate struct {
	DelayMs int          `json:"delay_ms"`
	Bids    [][2]float64 `json:"bids"`
	Asks    [][2]float64 `json:"asks"`
}

// InitialState is the starting book state.
type InitialState struct {
	Bids [][2]float64 `json:"bids"`
	Asks [][2]float64 `json:"asks"`
}

// Scenario is the full scenario file.
type Scenario struct {
	Initial InitialState     `json:"initial"`
	Updates []ScenarioUpdate `json:"updates"`
}

// LoadScenario reads and parses the scenario JSON file.
func LoadScenario(path string) (*Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read scenario file %q: %w", path, err)
	}
	var s Scenario
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("cannot parse scenario file %q: %w", path, err)
	}
	return &s, nil
}

func toLevels(s [][2]float64) []protocol.Level {
	if len(s) == 0 {
		return []protocol.Level{}
	}
	out := make([]protocol.Level, len(s))
	for i, v := range s {
		out[i] = protocol.Level{Price: v[0], Amount: v[1]}
	}
	return out
}

// RunSim runs the simulator in an infinite loop (circular scenario replay).
// It blocks until ctx is cancelled.
func RunSim(ctx context.Context, scenario *Scenario, ob *orderbook.OrderBook, h *hub.Hub, speed float64) {
	var seq int64
	for {
		total := len(scenario.Updates)
		for i, update := range scenario.Updates {
			select {
			case <-ctx.Done():
				return
			default:
			}

			if speed > 0 && update.DelayMs > 0 {
				delay := time.Duration(float64(update.DelayMs)/speed) * time.Millisecond
				timer := time.NewTimer(delay)
				select {
				case <-ctx.Done():
					timer.Stop()
					return
				case <-timer.C:
				}
			}

			ob.ApplyUpdate(update.Bids, update.Asks)
			seq++

			msg := protocol.DepthUpdate{
				Event: "depthUpdate",
				Seq:   seq,
				Ts:    time.Now().UnixMilli(),
				Bids:  toLevels(update.Bids),
				Asks:  toLevels(update.Asks),
			}
			data, err := json.Marshal(msg)
			if err != nil {
				log.Printf("sim marshal error update %d: %v", i+1, err)
				continue
			}
			h.Broadcast(data)
		}
		log.Printf("sim: scenario loop complete (%d updates), replaying...", total)
	}
}
