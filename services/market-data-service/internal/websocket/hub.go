package websocket

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gofiber/websocket/v2"
	"github.com/Rohianon/equishare-global-trading/pkg/alpaca"
	"github.com/Rohianon/equishare-global-trading/pkg/logger"
)

// Message types
const (
	MsgTypeSubscribe   = "subscribe"
	MsgTypeUnsubscribe = "unsubscribe"
	MsgTypeQuote       = "quote"
	MsgTypeTrade       = "trade"
	MsgTypeBar         = "bar"
	MsgTypeError       = "error"
	MsgTypePing        = "ping"
	MsgTypePong        = "pong"
)

// ClientMessage represents a message from the client
type ClientMessage struct {
	Type    string   `json:"type"`
	Symbols []string `json:"symbols,omitempty"`
}

// ServerMessage represents a message to the client
type ServerMessage struct {
	Type      string      `json:"type"`
	Symbol    string      `json:"symbol,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// QuoteData represents real-time quote data
type QuoteData struct {
	Symbol    string  `json:"symbol"`
	BidPrice  float64 `json:"bid_price"`
	BidSize   int     `json:"bid_size"`
	AskPrice  float64 `json:"ask_price"`
	AskSize   int     `json:"ask_size"`
	MidPrice  float64 `json:"mid_price"`
	Spread    float64 `json:"spread"`
	Timestamp string  `json:"timestamp"`
}

// Client represents a WebSocket client connection
type Client struct {
	ID          string
	Conn        *websocket.Conn
	Hub         *Hub
	Symbols     map[string]bool
	Send        chan []byte
	mu          sync.RWMutex
	lastPing    time.Time
}

// Hub manages all WebSocket clients and broadcasts
type Hub struct {
	clients      map[*Client]bool
	symbols      map[string]map[*Client]bool // symbol -> clients subscribed
	broadcast    chan *ServerMessage
	register     chan *Client
	unregister   chan *Client
	alpaca       alpaca.TradingClient
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	tickInterval time.Duration
}

// NewHub creates a new WebSocket hub
func NewHub(alpacaClient alpaca.TradingClient) *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	return &Hub{
		clients:      make(map[*Client]bool),
		symbols:      make(map[string]map[*Client]bool),
		broadcast:    make(chan *ServerMessage, 256),
		register:     make(chan *Client),
		unregister:   make(chan *Client),
		alpaca:       alpacaClient,
		ctx:          ctx,
		cancel:       cancel,
		tickInterval: 5 * time.Second, // Poll for quotes every 5 seconds
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	// Start the quote polling goroutine
	go h.pollQuotes()

	for {
		select {
		case <-h.ctx.Done():
			return

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			logger.Info().Str("client_id", client.ID).Msg("WebSocket client connected")

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
				// Remove from all symbol subscriptions
				for symbol := range client.Symbols {
					if clients, ok := h.symbols[symbol]; ok {
						delete(clients, client)
						if len(clients) == 0 {
							delete(h.symbols, symbol)
						}
					}
				}
			}
			h.mu.Unlock()
			logger.Info().Str("client_id", client.ID).Msg("WebSocket client disconnected")

		case message := <-h.broadcast:
			h.mu.RLock()
			// Only send to clients subscribed to this symbol
			if clients, ok := h.symbols[message.Symbol]; ok {
				data, err := json.Marshal(message)
				if err != nil {
					logger.Error().Err(err).Msg("Failed to marshal broadcast message")
					h.mu.RUnlock()
					continue
				}
				for client := range clients {
					select {
					case client.Send <- data:
					default:
						// Client buffer full, skip
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Stop gracefully shuts down the hub
func (h *Hub) Stop() {
	h.cancel()
}

// pollQuotes periodically fetches quotes for subscribed symbols
func (h *Hub) pollQuotes() {
	ticker := time.NewTicker(h.tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			h.fetchAndBroadcastQuotes()
		}
	}
}

// fetchAndBroadcastQuotes fetches quotes for all subscribed symbols and broadcasts them
func (h *Hub) fetchAndBroadcastQuotes() {
	h.mu.RLock()
	symbols := make([]string, 0, len(h.symbols))
	for symbol := range h.symbols {
		symbols = append(symbols, symbol)
	}
	h.mu.RUnlock()

	if len(symbols) == 0 {
		return
	}

	// Fetch quotes in batches of 100
	batchSize := 100
	for i := 0; i < len(symbols); i += batchSize {
		end := i + batchSize
		if end > len(symbols) {
			end = len(symbols)
		}
		batch := symbols[i:end]

		quotes, err := h.alpaca.GetMultiQuotes(h.ctx, batch)
		if err != nil {
			logger.Error().Err(err).Strs("symbols", batch).Msg("Failed to fetch quotes")
			continue
		}

		for symbol, quote := range quotes {
			midPrice := (quote.BidPrice + quote.AskPrice) / 2
			spread := quote.AskPrice - quote.BidPrice

			msg := &ServerMessage{
				Type:      MsgTypeQuote,
				Symbol:    symbol,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Data: QuoteData{
					Symbol:    symbol,
					BidPrice:  quote.BidPrice,
					BidSize:   quote.BidSize,
					AskPrice:  quote.AskPrice,
					AskSize:   quote.AskSize,
					MidPrice:  midPrice,
					Spread:    spread,
					Timestamp: quote.Timestamp,
				},
			}

			select {
			case h.broadcast <- msg:
			default:
				// Broadcast channel full, skip
			}
		}
	}
}

// Subscribe adds symbols to a client's subscription
func (h *Hub) Subscribe(client *Client, symbols []string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, symbol := range symbols {
		if _, ok := h.symbols[symbol]; !ok {
			h.symbols[symbol] = make(map[*Client]bool)
		}
		h.symbols[symbol][client] = true
		client.Symbols[symbol] = true
	}

	logger.Info().Str("client_id", client.ID).Strs("symbols", symbols).Msg("Client subscribed")
}

// Register adds a client to the hub
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unsubscribe removes symbols from a client's subscription
func (h *Hub) Unsubscribe(client *Client, symbols []string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, symbol := range symbols {
		if clients, ok := h.symbols[symbol]; ok {
			delete(clients, client)
			if len(clients) == 0 {
				delete(h.symbols, symbol)
			}
		}
		delete(client.Symbols, symbol)
	}

	logger.Info().Str("client_id", client.ID).Strs("symbols", symbols).Msg("Client unsubscribed")
}

// NewClient creates a new WebSocket client
func NewClient(id string, conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		ID:       id,
		Conn:     conn,
		Hub:      hub,
		Symbols:  make(map[string]bool),
		Send:     make(chan []byte, 256),
		lastPing: time.Now(),
	}
}

// ReadPump reads messages from the WebSocket connection
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Error().Err(err).Str("client_id", c.ID).Msg("WebSocket read error")
			}
			break
		}

		// Reset read deadline on any message
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		var msg ClientMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			c.sendError("Invalid message format")
			continue
		}

		switch msg.Type {
		case MsgTypeSubscribe:
			if len(msg.Symbols) > 0 {
				c.Hub.Subscribe(c, msg.Symbols)
			}
		case MsgTypeUnsubscribe:
			if len(msg.Symbols) > 0 {
				c.Hub.Unsubscribe(c, msg.Symbols)
			}
		case MsgTypePing:
			c.lastPing = time.Now()
			c.sendPong()
		default:
			c.sendError("Unknown message type: " + msg.Type)
		}
	}
}

// WritePump writes messages to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) sendError(errMsg string) {
	msg := ServerMessage{
		Type:      MsgTypeError,
		Error:     errMsg,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	data, _ := json.Marshal(msg)
	select {
	case c.Send <- data:
	default:
	}
}

func (c *Client) sendPong() {
	msg := ServerMessage{
		Type:      MsgTypePong,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	data, _ := json.Marshal(msg)
	select {
	case c.Send <- data:
	default:
	}
}
