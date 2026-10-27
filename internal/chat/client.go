package chat

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second    // max time to write one message
	pongWait       = 60 * time.Second    // max time without hearing from the client
	pingPeriod     = (pongWait * 9) / 10 // ping a bit before pongWait runs out
	maxMessageSize = 512                 // bytes; bigger messages close the connection
	sendBuffer     = 256                 // how many messages can wait in the mailbox
)

// Client is one connected browser tab.
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte // mailbox: hub puts messages here, writePump delivers them
	username string      // set once in NewClient, never changes: safe to read from any goroutine
	room     string      // current room ("" = none). ONLY the hub goroutine reads or writes this
	logger   *slog.Logger
}

func NewClient(hub *Hub, conn *websocket.Conn, username string, logger *slog.Logger) *Client {
	return &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, sendBuffer),
		username: username,
		logger:   logger,
	}
}

// Start runs the two pumps. Gorilla allows one reader and one writer
// per connection at a time, so each gets its own goroutine.
// From here on the pumps own the connection: they are the ones that close it.
func (c *Client) Start() {
	go c.writePump()
	go c.readPump()
}

// readPump: browser -> server -> hub.
// It ends when the browser leaves, a deadline passes, or the conn is closed.
func (c *Client) readPump() {
	defer func() {
		c.hub.Unregister(c) // hub removes us and closes c.send -> writePump stops
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		// Client answered our ping: it's alive, give it more time.
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			// Normal closes (tab closed, navigated away) are not worth a warning.
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Warn("read error", "username", c.username, "err", err)
			}
			return
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			c.logger.Debug("bad json", "username", c.username, "err", err)
			continue // one bad message isn't a reason to disconnect
		}
		if err := msg.validate(); err != nil {
			c.logger.Debug("invalid message", "username", c.username, "err", err)
			continue
		}

		// Never trust the name the browser sends: anyone could type someone else's.
		msg.Username = c.username

		// Don't read c.room here: only the hub goroutine owns it (that would be a data race).
		// The hub looks up our room when it handles the message.
		c.hub.Send(c, msg)
	}
}

// writePump: hub -> mailbox -> browser. Also sends pings.
// It is the only goroutine that writes to the connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close() // makes ReadMessage fail, so readPump stops too
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the mailbox: say goodbye and stop.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
