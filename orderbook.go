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
func (ob *OrderBook) GetSnapshot() (bids, asks [][2]float64) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	bids = make([][2]float64, 0, len(ob.bids))
	for price, qty := range ob.bids {
		bids = append(bids, [2]float64{price, qty})
	}
	sort.Slice(bids, func(i, j int) bool {
		return bids[i][0] > bids[j][0]
	})

	asks = make([][2]float64, 0, len(ob.asks))
	for price, qty := range ob.asks {
		asks = append(asks, [2]float64{price, qty})
	}
	sort.Slice(asks, func(i, j int) bool {
		return asks[i][0] < asks[j][0]
	})

	return bids, asks
}
