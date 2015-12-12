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

func apiUrl(path string, args ...interface{}) string {
	return fmt.Sprintf("https://api.stockfighter.io/ob/api/"+path, args...)
}

func gmUrl(path string, args ...interface{}) string {
	return fmt.Sprintf("https://api.stockfighter.io/gm/"+path, args...)
}

func wsUrl(path string, args ...interface{}) string {
	return fmt.Sprintf("wss://api.stockfighter.io/ob/ws/"+path, args...)
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

type orderRequest struct {
	Account   string    `json:"account"`
	Venue     string    `json:"venue"`
	Stock     string    `json:"stock"`
	Price     uint64    `json:"price"`
	Quantity  uint64    `json:"qty"`
	Direction string    `json:"direction"`
	OrderType OrderType `json:"market"`
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
	Ok bool
	Quote
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

// Check the API Is Up. Returns nil if ok, otherwise the error indicates the problem.
func (sf *Stockfighter) Heartbeat() error {
	var resp response
	return sf.do("GET", apiUrl("heartbeat"), nil, &resp)
}

// Check a venue is up. Returns nil if ok, otherwise the error indicates the problem.
func (sf *Stockfighter) VenueHeartbeat(venue string) error {
	var resp venueResponse
	if err := sf.do("GET", apiUrl("venues/%s/heartbeat", venue), nil, &resp); err != nil {
		return err
	}
	return nil
}

// Get the stocks available for trading on a venue.
func (sf *Stockfighter) Stocks(venue string) ([]Symbol, error) {
	var resp stocksResponse
	if err := sf.do("GET", apiUrl("venues/%s/stocks", venue), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Symbols, nil
}

// Get the orderbook for a particular stock.
func (sf *Stockfighter) OrderBook(venue, stock string) (*OrderBook, error) {
	var resp orderBookResponse
	if err := sf.do("GET", apiUrl("venues/%s/stocks/%s", venue, stock), nil, &resp); err != nil {
		return nil, err
	}
	return &resp.OrderBook, nil
}

// Get a quick look at the most recent trade information for a stock.
func (sf *Stockfighter) Quote(venue, stock string) (*Quote, error) {
	var resp quoteResponse
	if err := sf.do("GET", apiUrl("venues/%s/stocks/%s/quote", venue, stock), nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Quote, nil
}

// Place an order to buy a stock.
func (sf *Stockfighter) Buy(account, venue, stock string, price, quantity uint64, orderType OrderType) (*OrderState, error) {
	return sf.placeOrder(account, venue, stock, "buy", price, quantity, orderType)
}

// Place an order to sell a stock.
func (sf *Stockfighter) Sell(account, venue, stock string, price, quantity uint64, orderType OrderType) (*OrderState, error) {
	return sf.placeOrder(account, venue, stock, "sell", price, quantity, orderType)
}

// Get the status for an existing order.
func (sf *Stockfighter) Status(venue, stock string, id uint64) (*OrderState, error) {
	var resp orderResponse
	path := apiUrl("venues/%s/stocks/%s/orders/%d", venue, stock, id)
	if err := sf.do("GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.OrderState, nil
}

// Get the statuses for all an account's orders of a stock on a venue
func (sf *Stockfighter) StockStatus(account, venue, stock string) ([]OrderState, error) {
	path := apiUrl("venues/%s/accounts/%s/stocks/%s/orders", venue, account, stock)
	return sf.bulkStatus(path)
}

// Get the statuses for all an account's orders on a venue
func (sf *Stockfighter) VenueStatus(account, venue string) ([]OrderState, error) {
	path := apiUrl("venues/%s/accounts/%s/orders", venue, account)
	return sf.bulkStatus(path)
}

// Cancel an existing order
func (sf *Stockfighter) Cancel(venue, stock string, id uint64) error {
	var resp response
	path := apiUrl("venues/%s/stocks/%s/orders/%d", venue, stock, id)
	return sf.do("DELETE", path, nil, &resp)
}

// Subsribe to a stream of quotes for a venue. If stock is a non-empy string, only quotes for that stock are returned.
func (sf *Stockfighter) Quotes(account, venue, stock string) (chan *Quote, error) {
	path := wsUrl("%s/venues/%s/tickertape", account, venue)
	if len(stock) > 0 {
		path = wsUrl("%s/venues/%s/stocks/%s/tickertape", account, venue, stock)
	}
	c := make(chan *Quote)
	return c, sf.pump(path, func(conn *websocket.Conn) error {
		var quote quoteMessage
		if err := conn.ReadJSON(&quote); err != nil {
			close(c)
			return err
		}
		if quote.Ok {
			c <- &quote.Quote
		}
		return nil
	})
}

// Subsribe to a stream of executions for a venue. If stock is a non-empy string, only executions for that stock are returned.
func (sf *Stockfighter) Executions(account, venue, stock string) (chan *Execution, error) {
	path := wsUrl("%s/venues/%s/executions", account, venue)
	if len(stock) > 0 {
		path = wsUrl("%s/venues/%s/stocks/%s/executions", account, venue, stock)
	}
	c := make(chan *Execution)
	return c, sf.pump(path, func(conn *websocket.Conn) error {
		var execution executionMessage
		if err := conn.ReadJSON(&execution); err != nil {
			close(c)
			return err
		}
		if execution.Ok {
			c <- &execution.Execution
		}
		return nil
	})
}

// Start a new level.
func (sf *Stockfighter) Start(level string) (*Game, error) {
	var resp gameResponse
	if err := sf.do("POST", gmUrl("levels/%s", level), nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Game, nil
}

// Restart a level using the instance id from a previously started Game.
func (sf *Stockfighter) Restart(id uint64) error {
	var resp response
	return sf.do("POST", gmUrl("instances/%d/restart", id), nil, &resp)
}

// Resume a level using the instance id from a previously started Game.
func (sf *Stockfighter) Resume(id uint64) error {
	var resp response
	return sf.do("POST", gmUrl("instances/%d/resume", id), nil, &resp)
}

// Stop a level using the instance id from a previously started Game.
func (sf *Stockfighter) Stop(id uint64) error {
	var resp response
	return sf.do("POST", gmUrl("instances/%d/resume", id), nil, &resp)
}

// Get the GameState using the instance id from a previously started Game.
func (sf *Stockfighter) GameStatus(id uint64) (*GameState, error) {
	var resp gameStateResponse
	if err := sf.do("Get", gmUrl("instances/%d", id), nil, &resp); err != nil {
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

func (sf *Stockfighter) placeOrder(account, venue, stock, direction string, price, quantity uint64, orderType OrderType) (*OrderState, error) {
	order, err := encodeJson(&orderRequest{
		Account:   account,
		Venue:     venue,
		Stock:     stock,
		Direction: direction,
		Price:     price,
		Quantity:  quantity,
		OrderType: orderType,
	})
	if err != nil {
		return nil, err
	}
	var resp orderResponse
	if err := sf.do("POST", apiUrl("venues/%s/stocks/%s/orders", venue, stock), order, &resp); err != nil {
		return nil, err
	}
	return &resp.OrderState, nil
}

func (sf *Stockfighter) bulkStatus(url string) ([]OrderState, error) {
	var resp bulkOrderResponse
	if err := sf.do("GET", url, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Orders, nil
}

func (sf *Stockfighter) do(method, url string, body io.Reader, value apiCall) error {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return err
	}
	req.Header.Add("X-Starfighter-Authorization", sf.apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if sf.debug {
		out, _ := httputil.DumpRequest(req, true)
		log.Println(string(out))
		out, _ = httputil.DumpResponse(resp, true)
		log.Println(string(out))
	}
	if err := json.NewDecoder(resp.Body).Decode(value); err != nil {
		return err
	}
	if err := value.Err(); err != nil {
		return err
	}
	return nil
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
