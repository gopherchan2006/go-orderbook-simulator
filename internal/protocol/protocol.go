package protocol

// Level represents a single price level in the order book.
type Level struct {
	Price  float64 `json:"p"`
	Amount float64 `json:"q"`
}

// DepthSnapshot is sent once to a new subscriber with the full book state.
type DepthSnapshot struct {
	Event string  `json:"e"`
	Bids  []Level `json:"bids"`
	Asks  []Level `json:"asks"`
}

// DepthUpdate is a diff message: qty=0 means remove the level.
type DepthUpdate struct {
	Event string  `json:"e"`
	Seq   int64   `json:"seq"`
	Ts    int64   `json:"ts"`
	Bids  []Level `json:"b"`
	Asks  []Level `json:"a"`
}
