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
	Balances             map[string]int64
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
	OrderType        OrderType
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

type Evidence struct {
	Account          string `json:"account"`
	ExplanationLink  string `json:"explanation_link"`
	ExecutiveSummary string `json:"executive_summary"`
}

var orderTypes = [...]string{
	Limit:             "limit",
	Market:            "market",
	FillOrKill:        "fill-or-kill",
	ImmediateOrCancel: "immediate-or-cancel",
}

func ft(ts time.Time) string {
	return ts.Format(time.StampNano)
}

func (g Game) String() string {
	return fmt.Sprintf("Account: %s Venues: %+v Tickers: %+v InstanceId: %6d SecondsPerDay: %d", g.Account, g.Venues, g.Tickers, g.InstanceId, g.SecondsPerTradingDay)
}

func (os OrderState) String() string {
	format := "%s Venue: %s Symbol: %s Direction: %4s Price: %8d Quantity: %6d Filled: %6d/%6d Open: %5t Fills: %4d Type: %s"
	return fmt.Sprintf(format, ft(os.Timestamp), os.Venue, os.Symbol, os.Direction, os.Price, os.OriginalQuantity, os.TotalFilled, os.Quantity, os.Open, len(os.Fills), os.OrderType)
}

func (q Quote) String() string {
	format := "%s Venue: %s Symbol: %s Bid: %8d BidSize: %6d BidDepth %6d Ask: %8d AskSize: %6d AskDepth %6d Last: (%8d,%8d,%s)"
	return fmt.Sprintf(format, ft(q.QuoteTime), q.Venue, q.Symbol, q.Bid, q.BidSize, q.BidDepth, q.Ask, q.AskSize, q.AskDepth, q.Last, q.LastSize, ft(q.LastTrade))
}

func (e Execution) String() string {
	format := "%s Account: %s Venue: %s Symbol: %s Direction: %4s Price: %8d Filled: %6d Open: %5t Fills: %4d Standing: %6d Incoming: %6d Type: %s"
	return fmt.Sprintf(format, ft(e.FilledAt), e.Account, e.Venue, e.Symbol, e.Order.Direction, e.Price, e.Filled, e.Order.Open, len(e.Order.Fills), e.StandingId, e.IncomingId, e.Order.OrderType)
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
