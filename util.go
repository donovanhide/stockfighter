package stockfighter

import (
	"fmt"
	"time"
)

func Format(value interface{}) string {
	var (
		f    = "%+v"
		args = []interface{}{value}
	)
	switch v := value.(type) {
	case time.Time:
		return v.Format(time.StampNano)
	case *Quote:
		if v.Ask > 0 {
			f = "%s Venue: %s Symbol: %s Ask: %6d Quantity: %6d Depth: %6d Last: (%6d,%6d,%s)"
			args = []interface{}{Format(v.QuoteTime), v.Venue, v.Symbol, v.Ask, v.AskSize, v.AskDepth, v.Last, v.LastSize, Format(v.LastTrade)}
		} else {
			f = "%s Venue: %s Symbol: %s Bid: %6d Quantity: %6d Depth: %6d Last: (%6d,%6d,%s)"
			args = []interface{}{Format(v.QuoteTime), v.Venue, v.Symbol, v.Bid, v.BidSize, v.BidDepth, v.Last, v.LastSize, Format(v.LastTrade)}
		}
	case *Execution:
		f = "%s Venue: %s Symbol: %s Price: %6d Quantity: %6d Account: %s Id: %6d Incoming: %t Standing: %t"
		args = []interface{}{Format(v.FilledAt), v.Venue, v.Symbol, v.Price, v.Filled, v.Account, v.StandingId, v.IncomingComplete, v.StandingComplete}
	}
	return fmt.Sprintf(f, args...)
}
