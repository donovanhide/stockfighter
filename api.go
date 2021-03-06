// Package stockfighter provides a simple wrapper for the Stockfighter API:
//
// https://www.stockfighter.io/
//
// https://starfighter.readme.io/
package stockfighter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"

	"github.com/gorilla/websocket"
)

// Domain which hosts the API servers
var URL_BASE = "api.stockfighter.io"

func gmUrl(path string, args ...interface{}) string {
	format := "https://" + URL_BASE + "/gm/" + path
	return fmt.Sprintf(format, args...)
}

func apiUrl(path string, args ...interface{}) string {
	format := "https://" + URL_BASE + "/ob/api/" + path
	return fmt.Sprintf(format, args...)
}

func wsUrl(path string, args ...interface{}) string {
	format := "wss://" + URL_BASE + "/ob/api/ws/" + path
	return fmt.Sprintf(format, args...)
}

type apiCall interface {
	Err() error
}

type response struct {
	Ok    bool
	Error string
}

func (r response) Err() error {
	if len(r.Error) > 0 {
		return fmt.Errorf(r.Error)
	}
	return nil
}

type venueResponse struct {
	response
	Venue string
}

type stocksResponse struct {
	response
	Symbols []Symbol
}

type orderBookResponse struct {
	response
	OrderBook
}

type quoteResponse struct {
	response
	Quote
}

type orderResponse struct {
	response
	OrderState
}

type bulkOrderResponse struct {
	response
	Venue  string
	Orders []OrderState
}

type quoteMessage struct {
	Ok    bool
	Quote Quote
}

type executionMessage struct {
	Ok bool
	Execution
}

type gameResponse struct {
	response
	Game
}

type gameStateResponse struct {
	response
	GameState
}

type Stockfighter struct {
	apiKey string
	debug  bool
}

// Create new Stockfighter API instance.
// If debug is true, log all HTTP requests and responses.
func NewStockfighter(apiKey string, debug bool) *Stockfighter {
	return &Stockfighter{
		apiKey: apiKey,
		debug:  debug,
	}
}

// Check the API Is Up. If venue is a non-empty string, then check that venue.
// Returns nil if ok, otherwise the error indicates the problem.
func (sf *Stockfighter) Heartbeat(venue string) error {
	var resp response
	url := apiUrl("heartbeat")
	if len(venue) > 0 {
		url = apiUrl("venues/%s/heartbeat", venue)
	}
	return sf.do("GET", url, nil, &resp)
}

// Get the stocks available for trading on a venue.
func (sf *Stockfighter) Stocks(venue string) ([]Symbol, error) {
	var resp stocksResponse
	url := apiUrl("venues/%s/stocks", venue)
	if err := sf.do("GET", url, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Symbols, nil
}

// Get the orderbook for a particular stock.
func (sf *Stockfighter) OrderBook(venue, stock string) (*OrderBook, error) {
	var resp orderBookResponse
	url := apiUrl("venues/%s/stocks/%s", venue, stock)
	if err := sf.do("GET", url, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.OrderBook, nil
}

// Get a quick look at the most recent trade information for a stock.
func (sf *Stockfighter) Quote(venue, stock string) (*Quote, error) {
	var resp quoteResponse
	url := apiUrl("venues/%s/stocks/%s/quote", venue, stock)
	if err := sf.do("GET", url, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Quote, nil
}

// Place an order
func (sf *Stockfighter) Place(order *Order) (*OrderState, error) {
	body, err := encodeJson(order)
	if err != nil {
		return nil, err
	}
	var resp orderResponse
	url := apiUrl("venues/%s/stocks/%s/orders", order.Venue, order.Stock)
	if err := sf.do("POST", url, body, &resp); err != nil {
		return nil, err
	}
	return &resp.OrderState, nil
}

// Get the status for an existing order.
func (sf *Stockfighter) Status(venue, stock string, id uint64) (*OrderState, error) {
	var resp orderResponse
	url := apiUrl("venues/%s/stocks/%s/orders/%d", venue, stock, id)
	if err := sf.do("GET", url, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.OrderState, nil
}

// Cancel an existing order.
func (sf *Stockfighter) Cancel(venue, stock string, id uint64) (*OrderState, error) {
	var resp orderResponse
	url := apiUrl("venues/%s/stocks/%s/orders/%d", venue, stock, id)
	if err := sf.do("DELETE", url, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.OrderState, nil
}

// Get the statuses for all an account's orders of a stock on a venue.
// If stock is a non-empty string, only statuses for that stock are returned
func (sf *Stockfighter) StockStatus(account, venue, stock string) ([]OrderState, error) {
	url := apiUrl("venues/%s/accounts/%s/orders", venue, account)
	if len(stock) > 0 {
		url = apiUrl("venues/%s/accounts/%s/stocks/%s/orders", venue, account, stock)
	}
	var resp bulkOrderResponse
	if err := sf.do("GET", url, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Orders, nil
}

// Subscribe to a stream of quotes for a venue.
// If stock is a non-empy string, only quotes for that stock are returned.
func (sf *Stockfighter) Quotes(account, venue, stock string) (chan *Quote, error) {
	url := wsUrl("%s/venues/%s/tickertape", account, venue)
	if len(stock) > 0 {
		url = wsUrl("%s/venues/%s/tickertape/stocks/%s", account, venue, stock)
	}
	c := make(chan *Quote)
	return c, sf.pump(url, func(conn *websocket.Conn) error {
		var quote quoteMessage
		if err := sf.decodeMessage(conn, &quote); err != nil {
			close(c)
			return err
		}
		if quote.Ok {
			c <- &quote.Quote
		}
		return nil
	})
}

// Subscribe to a stream of executions for a venue.
// If stock is a non-empy string, only executions for that stock are returned.
func (sf *Stockfighter) Executions(account, venue, stock string) (chan *Execution, error) {
	url := wsUrl("%s/venues/%s/executions", account, venue)
	if len(stock) > 0 {
		url = wsUrl("%s/venues/%s/executions/stocks/%s", account, venue, stock)
	}
	c := make(chan *Execution)
	return c, sf.pump(url, func(conn *websocket.Conn) error {
		var execution executionMessage
		if err := sf.decodeMessage(conn, &execution); err != nil {
			close(c)
			return err
		}
		if execution.Ok {
			c <- &execution.Execution
		}
		return nil
	})
}

func (sf *Stockfighter) decodeMessage(conn *websocket.Conn, v interface{}) error {
	_, msg, err := conn.ReadMessage()
	if err != nil {
		return err
	}
	if sf.debug {
		log.Println(string(msg))
	}
	return json.Unmarshal(msg, v)
}

// Start a new level.
func (sf *Stockfighter) Start(level string) (*Game, error) {
	var resp gameResponse
	url := gmUrl("levels/%s", level)
	if err := sf.do("POST", url, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Game, nil
}

// Restart a level using the instance id from a previously started Game.
func (sf *Stockfighter) Restart(id uint64) error {
	var resp response
	url := gmUrl("instances/%d/restart", id)
	return sf.do("POST", url, nil, &resp)
}

// Resume a level using the instance id from a previously started Game.
func (sf *Stockfighter) Resume(id uint64) error {
	var resp response
	url := gmUrl("instances/%d/resume", id)
	return sf.do("POST", url, nil, &resp)
}

// Stop a level using the instance id from a previously started Game.
func (sf *Stockfighter) Stop(id uint64) error {
	var resp response
	url := gmUrl("instances/%d/stop", id)
	return sf.do("POST", url, nil, &resp)
}

// Get the GameState using the instance id from a previously started Game.
func (sf *Stockfighter) GameStatus(id uint64) (*GameState, error) {
	var resp gameStateResponse
	url := gmUrl("instances/%d", id)
	if err := sf.do("GET", url, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.GameState, nil
}

func (sf *Stockfighter) Judge(id uint64, evidence *Evidence) (*GameState, error) {
	body, err := encodeJson(evidence)
	if err != nil {
		return nil, err
	}
	var resp gameStateResponse
	url := gmUrl("instances/%d/judge", id)
	if err := sf.do("POST", url, body, &resp); err != nil {
		return nil, err
	}
	return &resp.GameState, nil

}

func encodeJson(v interface{}) (io.Reader, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		return nil, err
	}
	return &buf, nil
}

func (sf *Stockfighter) do(method, url string, body io.Reader, value apiCall) error {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return err
	}
	req.Header.Add("X-Starfighter-Authorization", sf.apiKey)
	if sf.debug {
		out, _ := httputil.DumpRequest(req, true)
		log.Println(string(out))
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if sf.debug {
		out, _ := httputil.DumpResponse(resp, true)
		log.Println(string(out))
	}
	if err := json.NewDecoder(resp.Body).Decode(value); err != nil {
		if resp.StatusCode >= 500 {
			return fmt.Errorf(resp.Status)
		}
		return err
	}
	return value.Err()
}

func (sf *Stockfighter) pump(url string, f func(*websocket.Conn) error) error {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return err
	}
	go func() {
		defer conn.Close()
		for err := f(conn); err == nil; err = f(conn) {
		}
	}()
	return nil
}
