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

// Side of a trade.
type Side string

const (
	Buy  Side = "buy"
	Sell Side = "sell"
)

// Trade represents a single executed trade from the tape.
type Trade struct {
	Event     string  `json:"e"`
	Ts        int64   `json:"ts"`
	Price     float64 `json:"p"`
	Qty       float64 `json:"q"`
	Side      Side    `json:"side"` // aggressor side: buy=market buy, sell=market sell
	Seq       int64   `json:"seq"`
}
