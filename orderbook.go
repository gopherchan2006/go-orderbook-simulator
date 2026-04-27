package main

import (
	"sort"
	"sync"
)

type OrderBook struct {
	mu   sync.RWMutex
	bids map[float64]float64
	asks map[float64]float64
}

func NewOrderBook() *OrderBook {
	return &OrderBook{
		bids: make(map[float64]float64),
		asks: make(map[float64]float64),
	}
}

func (ob *OrderBook) ApplyUpdate(bids, asks [][2]float64) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	for _, level := range bids {
		price, qty := level[0], level[1]
		if qty == 0 {
			delete(ob.bids, price)
		} else {
			ob.bids[price] = qty
		}
	}

	for _, level := range asks {
		price, qty := level[0], level[1]
		if qty == 0 {
			delete(ob.asks, price)
		} else {
			ob.asks[price] = qty
		}
	}
}
func (ob *OrderBook) GetSnapshot() (bids, asks []Level) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	bids = make([]Level, 0, len(ob.bids))
	for price, qty := range ob.bids {
		bids = append(bids, Level{Price: price, Amount: qty})
	}
	sort.Slice(bids, func(i, j int) bool {
		return bids[i].Price > bids[j].Price
	})

	asks = make([]Level, 0, len(ob.asks))
	for price, qty := range ob.asks {
		asks = append(asks, Level{Price: price, Amount: qty})
	}
	sort.Slice(asks, func(i, j int) bool {
		return asks[i].Price < asks[j].Price
	})

	return bids, asks
}
