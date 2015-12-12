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

const (
	apiUrl = "https://api.stockfighter.io/ob/api/"
	wsUrl  = "https://api.stockfighter.io/ob/ws/"
)

type apiCall interface {
	Err() error
}

type response struct {
	Ok    bool
	Error string
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

func (r response) Err() error {
	if len(r.Error) > 0 {
		return fmt.Errorf(r.Error)
	}
	return nil
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
	return sf.do("GET", "heartbeat", nil, &resp)
}

// Check a venue is up. Returns nil if ok, otherwise the error indicates the problem.
func (sf *Stockfighter) VenueHeartbeat(venue string) error {
	var resp venueResponse
	if err := sf.do("GET", "venues/"+venue+"/heartbeat", nil, &resp); err != nil {
		return err
	}
	return nil
}

// Get the stocks available for trading on a venue.
func (sf *Stockfighter) Stocks(venue string) ([]Symbol, error) {
	var resp stocksResponse
	if err := sf.do("GET", "venues/"+venue+"/stocks", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Symbols, nil
}

// Get the orderbook for a particular stock.
func (sf *Stockfighter) OrderBook(venue, stock string) (*OrderBook, error) {
	var resp orderBookResponse
	if err := sf.do("GET", "venues/"+venue+"/stocks/"+stock, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.OrderBook, nil
}

// Get a quick look at the most recent trade information for a stock.
func (sf *Stockfighter) Quote(venue, stock string) (*Quote, error) {
	var resp quoteResponse
	if err := sf.do("GET", "venues/"+venue+"/stocks/"+stock+"/quote", nil, &resp); err != nil {
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
	path := fmt.Sprintf("venues/%s/stocks/%s/orders/%d", venue, stock, id)
	if err := sf.do("GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.OrderState, nil
}

// Get the statuses for all an account's orders of a stock on a venue
func (sf *Stockfighter) StockStatus(account, venue, stock string) ([]OrderState, error) {
	path := fmt.Sprintf("venues/%s/accounts/%s/stocks/%s/orders", venue, account, stock)
	return sf.bulkStatus(path)
}

// Get the statuses for all an account's orders on a venue
func (sf *Stockfighter) VenueStatus(account, venue string) ([]OrderState, error) {
	path := fmt.Sprintf("venues/%s/accounts/%s/orders", venue, account)
	return sf.bulkStatus(path)
}

// Cancel an existing order
func (sf *Stockfighter) Cancel(venue, stock string, id uint64) error {
	var resp response
	path := fmt.Sprintf("venues/%s/stocks/%s/orders/%d", venue, stock, id)
	return sf.do("DELETE", path, nil, &resp)
}

// Subsribe to a stream of quotes for a venue. If stock is a non-empy string, only quotes for that stock are returned.
func (sf *Stockfighter) Quotes(account, venue, stock string) (chan *Quote, error) {
	path := fmt.Sprintf("%s/venues/%s/tickertape", account, venue)
	if len(stock) > 0 {
		path = fmt.Sprintf("%s/venues/%s/stocks/%s/tickertape", account, venue, stock)
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
func (sf *Stockfighter) VenueExecutions(account, venue, stock string) (chan *Execution, error) {
	path := fmt.Sprintf("%s/venues/%s/executions", account, venue)
	if len(stock) > 0 {
		path = fmt.Sprintf("%s/venues/%s/stocks/%s/executions", account, venue, stock)
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

func (sf *Stockfighter) placeOrder(account, venue, stock, direction string, price, quantity uint64, orderType OrderType) (*OrderState, error) {
	order := &orderRequest{
		Account:   account,
		Venue:     venue,
		Stock:     stock,
		Direction: direction,
		Price:     price,
		Quantity:  quantity,
		OrderType: orderType,
	}
	var resp orderResponse
	if err := sf.do("POST", "venues/"+venue+"/stocks/"+stock+"/orders", order, &resp); err != nil {
		return nil, err
	}
	if resp.Venue != venue || resp.Symbol != stock {
		return nil, fmt.Errorf("venue or stock in response does not match: %s %s %s %s", venue, resp.Venue, stock, resp.Symbol)
	}
	return &resp.OrderState, nil
}

func (sf *Stockfighter) bulkStatus(path string) ([]OrderState, error) {
	var resp bulkOrderResponse
	if err := sf.do("GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Orders, nil
}

func (sf *Stockfighter) do(method, path string, body interface{}, value apiCall) error {
	var r io.Reader
	if body != nil {
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return err
		}
		r = &buf
	}
	req, err := http.NewRequest(method, apiUrl+path, r)
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

func (sf *Stockfighter) pump(path string, f func(*websocket.Conn) error) error {
	conn, _, err := websocket.DefaultDialer.Dial(wsUrl+path, nil)
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
