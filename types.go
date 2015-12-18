package stockfighter

import (
	"fmt"
	"time"
)

type OrderType int

const (
	Limit OrderType = iota
	Market
	FillOrKill
	ImmediateOrCancel
)

type Game struct {
	Account              string
	InstanceId           uint64
	Instructions         map[string]string
	SecondsPerTradingDay uint64
	Tickers              []string
	Venues               []string
}

type GameState struct {
	Details struct {
		EndOfTheWorldDay uint64
		TradingDay       uint64
	}
	Flash struct {
		Info    string
		Warning string
		Danger  string
	}
	Done  bool
	Id    uint64
	State string
}

type Symbol struct {
	Symbol string
	Name   string
}

type Order struct {
	Account   string    `json:"account"`
	Venue     string    `json:"venue"`
	Stock     string    `json:"stock"`
	Price     uint64    `json:"price"`
	Quantity  uint64    `json:"qty"`
	Direction string    `json:"direction"`
	OrderType OrderType `json:"orderType"`
}

type StandingOrder struct {
	Price    uint64
	Quantity uint64 `json:"qty"`
	IsBuy    bool
}

type StandingOrderSlice []StandingOrder

type OrderBook struct {
	Venue     string
	Symbol    string
	Asks      StandingOrderSlice
	Bids      StandingOrderSlice
	TimeStamp time.Time `json:"ts"`
}

type Fill struct {
	Price     uint64
	Quantity  uint64    `json:"qty"`
	TimeStamp time.Time `json:"ts"`
}

type OrderState struct {
	Venue            string
	Symbol           string
	Price            uint64
	OriginalQuantity uint64 `json:"originalQty"`
	Quantity         uint64 `json:"qty"`
	Direction        string
	Type             OrderType
	Id               uint64
	Account          string
	Timestamp        time.Time `json:"ts"`
	Fills            []Fill
	TotalFilled      uint64
	Open             bool
}

type Quote struct {
	Venue     string
	Symbol    string
	Bid       uint64
	BidSize   uint64
	BidDepth  uint64
	Ask       uint64
	AskSize   uint64
	AskDepth  uint64
	Last      uint64
	LastSize  uint64
	LastTrade time.Time
	QuoteTime time.Time
}

type Execution struct {
	Account          string
	Venue            string
	Symbol           string
	Order            OrderState
	StandingId       uint64
	IncomingId       uint64
	Price            uint64
	Filled           uint64
	FilledAt         time.Time
	StandingComplete bool
	IncomingComplete bool
}

var orderTypes = [...]string{
	Limit:             "limit",
	Market:            "market",
	FillOrKill:        "fill-or-kill",
	ImmediateOrCancel: "immediate-or-cancel",
}

func (o OrderType) MarshalText() ([]byte, error) {
	return []byte(orderTypes[o]), nil
}

func (o OrderType) String() string {
	return orderTypes[o]
}

var orderTypeMap = map[string]OrderType{
	"limit":               Limit,
	"market":              Market,
	"fill-or-kill":        FillOrKill,
	"immediate-or-cancel": ImmediateOrCancel,
}

func (o *OrderType) UnmarshalText(text []byte) error {
	typ, ok := orderTypeMap[string(text)]
	if ok {
		*o = typ
		return nil
	}
	return fmt.Errorf("Unknown order type: %s", text)
}

// Total depth of offers
func (s StandingOrderSlice) Depth() uint64 {
	var depth uint64
	for _, so := range s {
		depth += so.Quantity
	}
	return depth
}
