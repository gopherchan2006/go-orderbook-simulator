package protocol

import "encoding/json"

// Level represents a single price level in the order book.
// It marshals/unmarshals as a JSON array [price, qty] to match frontend expectations.
type Level struct {
	Price  float64
	Amount float64
}

func (l Level) MarshalJSON() ([]byte, error) {
	return json.Marshal([2]float64{l.Price, l.Amount})
}

func (l *Level) UnmarshalJSON(data []byte) error {
	var arr [2]float64
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}
	l.Price = arr[0]
	l.Amount = arr[1]
	return nil
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
