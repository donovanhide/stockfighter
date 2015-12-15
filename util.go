package stockfighter

import (
	"fmt"
	"strings"
	"time"
)

func Depth(orders []StandingOrder) uint64 {
	var depth uint64
	for _, o := range orders {
		depth += o.Quantity
	}
	return depth
}

func Format(value interface{}) string {
	var (
		f    = "%+v"
		args = []interface{}{value}
	)
	switch v := value.(type) {
	case Fill:
		f = "%s Price: %6d Quantity: %6d"
		args = []interface{}{Format(v.TimeStamp), v.Price, v.Quantity}
	case []Fill:
		var s []string
		for _, fill := range v {
			s = append(s, Format(fill))
		}
		return strings.Join(s, "\n")
	case StandingOrder:
		f = "(%d,%d)"
		args = []interface{}{v.Price, v.Quantity}
	case []StandingOrder:
		var s []string
		for _, fill := range v {
			s = append(s, Format(fill))
		}
		return strings.Join(s, ",")
	case *OrderBook:
		f = "%s Venue: %s Symbol: %s Asks: [%s] AskDepth: %d Bids: [%s] BidDepth: %d"
		args = []interface{}{Format(v.TimeStamp), v.Venue, v.Symbol, Format(v.Asks), Depth(v.Asks), Format(v.Bids), Depth(v.Bids)}
	case *OrderState:
		f = "%s Venue: %s Symbol: %s Price: %6d Quantity: %6d Account: %s Id: %6d Direction: %s Type: %s\n%s"
		args = []interface{}{Format(v.Timestamp), v.Venue, v.Symbol, v.Price, v.Quantity, v.Account, v.Id, v.Direction, v.Type, Format(v.Fills)}
	case *Quote:
		f = "%s Venue: %s Symbol: %s Ask: %6d AskSize: %6d AskDepth: %6d Bid: %6d BidSize %6d BidDepth: %6d Last: (%6d,%6d,%s)"
		args = []interface{}{Format(v.QuoteTime), v.Venue, v.Symbol, v.Ask, v.AskSize, v.AskDepth, v.Bid, v.BidSize, v.BidDepth, v.Last, v.LastSize, Format(v.LastTrade)}
	case *Execution:
		f = "%s Venue: %s Symbol: %s Price: %6d Quantity: %6d Account: %s Id: %6d Incoming: %t Standing: %t\n%s"
		args = []interface{}{Format(v.FilledAt), v.Venue, v.Symbol, v.Price, v.Filled, v.Account, v.StandingId, v.IncomingComplete, v.StandingComplete, Format(&v.Order)}
	case time.Time:
		return v.Format(time.StampNano)
	}
	return fmt.Sprintf(f, args...)
}
