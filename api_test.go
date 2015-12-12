package stockfighter

import (
	"flag"
	"testing"
)

var (
	apiKey = flag.String("api_key", "", "set for testing authenticated api calls")
	debug  = flag.Bool("debug", false, "dump HTTP requests and responses")
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
	if len(*apiKey) == 0 {
		t.Skip("Skipping authenticated tests. Set api_key flag to run. This will start and stop a game for your account!")
	}

	sf := NewStockfighter(*apiKey, *debug)
	game, err := sf.Start("first_steps")
	checkErr(t, "Start", err)
	defer sf.Stop(game.InstanceId)

	if len(game.Venues) == 0 {
		t.Fatalf("No venues")
	}
	venue := game.Venues[0]

	if len(game.Tickers) == 0 {
		t.Fatalf("No tickers")
	}
	stock := game.Tickers[0]

	quotes, err := sf.Quotes(game.Account, venue, stock)
	checkErr(t, "Quotes", err)
	for i := 0; i < 5; i++ {
		t.Log(<-quotes)
	}
}
