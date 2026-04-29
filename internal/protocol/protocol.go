package protocol

import "encoding/json"

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

type DepthSnapshot struct {
	Event string  `json:"e"`
	Bids  []Level `json:"bids"`
	Asks  []Level `json:"asks"`
}

type DepthUpdate struct {
	Event string  `json:"e"`
	Seq   int64   `json:"seq"`
	Ts    int64   `json:"ts"`
	Bids  []Level `json:"b"`
	Asks  []Level `json:"a"`
}

type Side string

const (
	Buy  Side = "buy"
	Sell Side = "sell"
)

type Trade struct {
	Event string  `json:"e"`
	Ts    int64   `json:"ts"`
	Price float64 `json:"p"`
	Qty   float64 `json:"q"`
	Side  Side    `json:"side"`
	Seq   int64   `json:"seq"`
}
