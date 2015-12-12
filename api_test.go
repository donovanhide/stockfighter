package stockfighter

import (
	"flag"
	"testing"
)

var (
	apiKey  = flag.String("api_key", "", "set for testing authenticated api calls")
	account = flag.String("account", "", "account for placing and reviewing orders")
	debug   = flag.Bool("debug", false, "dump HTTP requests and responses")
)

func init() {
	flag.Parse()
}

func checkErr(t *testing.T, desc string, err error) {
	if err != nil {
		t.Fatalf("%s: %s", desc, err.Error())
	}
}

func TestUnauthenticated(t *testing.T) {
	sf := NewStockfighter(*apiKey, *debug)

	checkErr(t, "Heartbeat", sf.Heartbeat())
	checkErr(t, "Venue Heartbet", sf.VenueHeartbeat("TESTEX"))

	stocks, err := sf.Stocks("TESTEX")
	checkErr(t, "Stocks", err)
	if len(stocks) == 0 {
		t.Fatalf("No stocks returned")
	}

	orderbook, err := sf.OrderBook("TESTEX", "FOOBAR")
	checkErr(t, "Orderbook", err)
	if len(orderbook.Asks) == 0 && len(orderbook.Bids) == 0 {
		t.Fatalf("No asks or bids returned")
	}

	quote, err := sf.Quote("TESTEX", "FOOBAR")
	checkErr(t, "Quote", err)
	if quote.LastTrade.IsZero() {
		t.Fatalf("Invalid last trade")
	}
}

func TestAuthenticated(t *testing.T) {
	if len(*apiKey) == 0 || len(*account) == 0 {
		t.Skip("Skipping authenticated tests. Set account and api_key flags to run.")
	}

	sf := NewStockfighter(*apiKey, *debug)

	order, err := sf.Buy(*account, "TESTEX", "FOOBAR", 100, 100, Limit)
	checkErr(t, "Buy", err)
	if order.Account != *account {
		t.Fatalf("Wrong account: %s %s", order.Account, account)
	}
}
