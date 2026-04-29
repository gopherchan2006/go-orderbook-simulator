package orderbook

import (
	"sort"
	"sync"

	"go-orderbook-simulator/internal/protocol"
)

type OrderBook struct {
	mu   sync.RWMutex
	bids map[float64]float64
	asks map[float64]float64
}

func New() *OrderBook {
	return &OrderBook{
		bids: make(map[float64]float64),
		asks: make(map[float64]float64),
	}
}

func (ob *OrderBook) ApplyUpdate(bids, asks [][2]float64) {
	ob.mu.Lock()
	defer ob.mu.Unlock()
	for _, l := range bids {
		if l[1] == 0 {
			delete(ob.bids, l[0])
		} else {
			ob.bids[l[0]] = l[1]
		}
	}
	for _, l := range asks {
		if l[1] == 0 {
			delete(ob.asks, l[0])
		} else {
			ob.asks[l[0]] = l[1]
		}
	}
}

func (ob *OrderBook) ApplyLevels(bids, asks []protocol.Level) {
	ob.mu.Lock()
	defer ob.mu.Unlock()
	for _, l := range bids {
		if l.Amount == 0 {
			delete(ob.bids, l.Price)
		} else {
			ob.bids[l.Price] = l.Amount
		}
	}
	for _, l := range asks {
		if l.Amount == 0 {
			delete(ob.asks, l.Price)
		} else {
			ob.asks[l.Price] = l.Amount
		}
	}
}

func (ob *OrderBook) GetSnapshot() (bids, asks []protocol.Level) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	bids = make([]protocol.Level, 0, len(ob.bids))
	for p, q := range ob.bids {
		bids = append(bids, protocol.Level{Price: p, Amount: q})
	}
	sort.Slice(bids, func(i, j int) bool { return bids[i].Price > bids[j].Price })

	asks = make([]protocol.Level, 0, len(ob.asks))
	for p, q := range ob.asks {
		asks = append(asks, protocol.Level{Price: p, Amount: q})
	}
	sort.Slice(asks, func(i, j int) bool { return asks[i].Price < asks[j].Price })

	return
}

func (ob *OrderBook) BestBid() (price, qty float64) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	for p, q := range ob.bids {
		if p > price {
			price, qty = p, q
		}
	}
	return
}

func (ob *OrderBook) BestAsk() (price, qty float64) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	price = 1e18
	for p, q := range ob.asks {
		if p < price {
			price, qty = p, q
		}
	}
	if price == 1e18 {
		price = 0
	}
	return
}

func (ob *OrderBook) TopNImbalance(n int) float64 {
	bids, asks := ob.GetSnapshot()
	var bidSum, askSum float64
	for i := 0; i < n && i < len(bids); i++ {
		bidSum += bids[i].Amount
	}
	for i := 0; i < n && i < len(asks); i++ {
		askSum += asks[i].Amount
	}
	total := bidSum + askSum
	if total == 0 {
		return 0
	}
	return (bidSum - askSum) / total
}

func (ob *OrderBook) ConsumeAsks(qty float64) [][2]float64 {
	ob.mu.Lock()
	defer ob.mu.Unlock()
	return ob.consumeSide(ob.asks, qty, true)
}

func (ob *OrderBook) ConsumeBids(qty float64) [][2]float64 {
	ob.mu.Lock()
	defer ob.mu.Unlock()
	return ob.consumeSide(ob.bids, qty, false)
}

func (ob *OrderBook) consumeSide(side map[float64]float64, qty float64, ascending bool) [][2]float64 {
	prices := make([]float64, 0, len(side))
	for p := range side {
		prices = append(prices, p)
	}
	if ascending {
		sort.Float64s(prices)
	} else {
		sort.Sort(sort.Reverse(sort.Float64Slice(prices)))
	}

	var fills [][2]float64
	remaining := qty
	for _, p := range prices {
		if remaining <= 0 {
			break
		}
		avail := side[p]
		fill := avail
		if fill > remaining {
			fill = remaining
		}
		fills = append(fills, [2]float64{p, fill})
		remaining -= fill
		avail -= fill
		if avail <= 1e-10 {
			delete(side, p)
		} else {
			side[p] = avail
		}
	}
	return fills
}
