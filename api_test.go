package stockfighter

import (
	"flag"
	"testing"
	"time"
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

	checkErr(t, "Heartbeat", sf.Heartbeat(""))
	checkErr(t, "Venue Heartbeat", sf.Heartbeat("TESTEX"))

	stocks, err := sf.Stocks("TESTEX")
	checkErr(t, "Stocks", err)
	if len(stocks) == 0 {
		t.Fatalf("No stocks returned")
	}

	_, err = sf.OrderBook("TESTEX", "FOOBAR")
	checkErr(t, "Orderbook", err)

	_, err = sf.Quote("TESTEX", "FOOBAR")
	checkErr(t, "Quote", err)
}

func TestAuthenticated(t *testing.T) {
	if len(*apiKey) == 0 {
		t.Skip("Skipping authenticated tests. Set api_key flag to run. This will start and stop a game for your account!")
	}

	sf := NewStockfighter(*apiKey, *debug)
	game, err := sf.Start("first_steps")
	checkErr(t, "Start", err)

	t.Log(game)

	defer func() {
		checkErr(t, "Stop", sf.Stop(game.InstanceId))
	}()

	if len(game.Venues) == 0 {
		t.Fatalf("No venues")
	}

	if len(game.Tickers) == 0 {
		t.Fatalf("No tickers")
	}

	var (
		account = game.Account
		venue   = game.Venues[0]
		stock   = game.Tickers[0]
	)

	quotes, err := sf.Quotes(account, venue, stock)
	checkErr(t, "Quotes", err)

	executions, err := sf.Executions(account, venue, stock)
	checkErr(t, "Executions", err)

	// Loop until we get a filled market trade
	var ts time.Time
	for {
		os, err := sf.Place(&Order{
			Account:   account,
			Venue:     venue,
			Stock:     stock,
			Price:     0,
			Quantity:  100,
			Direction: "buy",
			OrderType: Market,
		})
		checkErr(t, "Place", err)
		t.Log(os)
		if len(os.Fills) > 0 {
			ts = os.Fills[0].TimeStamp
			break
		}
		time.Sleep(time.Second)
	}

	// Check trade appears in quote stream
	for quote := range quotes {
		if quote.LastTrade.Equal(ts) {
			t.Log(quote)
			break
		}
	}

	// Check trade appears in executionstream
	for execution := range executions {
		if execution.FilledAt.Equal(ts) {
			t.Log(execution)
			break
		}
	}

	orders, err := sf.StockStatus(account, venue, stock)
	checkErr(t, "StockStatus", err)

	if len(orders) == 0 {
		t.Fatalf("No record of order")
	}
	t.Log(orders[0])
}
