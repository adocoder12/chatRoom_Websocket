package chat

import (
	"context"
	"encoding/json"
	"log/slog"
)

// incoming is a message from a browser plus who sent it.
// The hub needs the sender to know which room the message goes to.
type incoming struct {
	from *Client
	msg  Message
}

// Hub is the receptionist: it knows who is connected and who is in which room.
// Only the Run goroutine reads or writes clients, rooms and Client.room,
// so no mutex is needed. Everyone else talks to the hub through channels.
type Hub struct {
	logger *slog.Logger

	clients map[*Client]bool            // everyone connected, in a room or not
	rooms   map[string]map[*Client]bool // room name -> people in that room

	register   chan *Client  // "a new client arrived"
	unregister chan *Client  // "a client left"
	incoming   chan incoming // "a browser sent something" (join / leave / message)
	done       chan struct{} // closed when Run stops, so callers never block forever
}

func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		logger:     logger,
		clients:    make(map[*Client]bool),
		rooms:      make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		incoming:   make(chan incoming),
		done:       make(chan struct{}),
	}
}

// Run is the hub's only goroutine. It handles one event at a time,
// so the maps are never touched by two goroutines at once.
func (h *Hub) Run(ctx context.Context) {
	defer close(h.done) // tell Register/Unregister/Send "the hub is gone"

	for {
		select {
		case <-ctx.Done():
			// Server is shutting down. server.Shutdown doesn't close WebSockets,
			// so we do it: closing each mailbox makes its writePump close the conn.
			// No "left the room" messages here: everyone is leaving at once.
			for c := range h.clients {
				close(c.send)
			}
			h.logger.Info("hub stopped", "closed", len(h.clients))
			return

		case c := <-h.register:
			h.clients[c] = true
			h.logger.Info("client connected to hub", "username", c.username, "online", len(h.clients))
			h.sendTo(c, Message{Type: TypeSystem, Content: "Welcome " + c.username + "! Join a room to start chatting."})

		case c := <-h.unregister:
			// Without this case, Unregister would block forever (goroutine leak).
			// The map check matters: a slow client may already be removed.
			if h.clients[c] {
				h.removeClient(c)
				h.logger.Info("client left", "username", c.username, "online", len(h.clients))
			}

		case in := <-h.incoming:
			h.handle(in.from, in.msg)
		}
	}
}

// handle decides what to do with a message from a browser.
func (h *Hub) handle(c *Client, msg Message) {
	if !h.clients[c] {
		return // client was already removed (e.g. dropped as slow)
	}

	switch msg.Type {
	case TypeJoin:
		h.joinRoom(c, msg.Room)

	case TypeLeave:
		h.leaveRoom(c)

	case TypeMessage:
		if c.room == "" {
			h.sendTo(c, Message{Type: TypeSystem, Content: "Join a room first."})
			return
		}
		// The room comes from the hub, not the browser: you can only
		// talk in the room you actually joined.
		msg.Room = c.room
		h.broadcast(c.room, msg)
	}
}

func (h *Hub) joinRoom(c *Client, room string) {
	if c.room == room {
		return // already there
	}
	h.leaveRoom(c) // one room at a time

	if h.rooms[room] == nil {
		h.rooms[room] = make(map[*Client]bool) // first person creates the room
	}
	h.rooms[room][c] = true
	c.room = room

	h.logger.Info("joined room", "username", c.username, "room", room, "in_room", len(h.rooms[room]))
	h.broadcast(room, Message{Type: TypeJoin, Room: room, Content: c.username + " joined " + room})
}

func (h *Hub) leaveRoom(c *Client) {
	room := c.room
	if room == "" {
		return // not in a room
	}

	delete(h.rooms[room], c)
	c.room = ""
	if len(h.rooms[room]) == 0 {
		delete(h.rooms, room) // last person out: the room disappears
	}

	h.logger.Info("left room", "username", c.username, "room", room)
	h.broadcast(room, Message{Type: TypeLeave, Room: room, Content: c.username + " left " + room})
}

// broadcast sends msg to everyone in room.
func (h *Hub) broadcast(room string, msg Message) {
	data, err := json.Marshal(msg) // encode once, not once per client
	if err != nil {
		h.logger.Error("encode message", "err", err)
		return
	}

	// Collect slow clients and remove them AFTER the loop.
	// Removing one sends a "left" message to this same room,
	// which would start a new loop inside this one.
	var slow []*Client
	for c := range h.rooms[room] {
		select {
		case c.send <- data:
		default:
			// Mailbox full: this client is too slow. Drop it so it
			// can't block everyone else.
			slow = append(slow, c)
		}
	}
	for _, c := range slow {
		h.logger.Warn("dropped slow client", "username", c.username)
		h.removeClient(c)
	}
}

// sendTo sends msg to one client only (welcome text, errors).
func (h *Hub) sendTo(c *Client, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error("encode message", "err", err)
		return
	}
	select {
	case c.send <- data:
	default:
		// mailbox full: skip, the next broadcast will drop it
	}
}

// removeClient takes the client out of its room and the hub, then closes its mailbox.
// Only call it for a client still in h.clients: closing a channel twice panics.
func (h *Hub) removeClient(c *Client) {
	if !h.clients[c] {
		return
	}
	delete(h.clients, c) // first, so nothing below sends to it again
	h.leaveRoom(c)
	close(c.send) // the hub is the only sender on c.send, so the hub closes it
}

// The methods below are how other goroutines talk to the hub.
// Each one also listens on h.done: if the hub already stopped,
// nobody is receiving, and a plain send would block forever.

func (h *Hub) Register(c *Client) {
	select {
	case h.register <- c:
	case <-h.done:
	}
}

func (h *Hub) Unregister(c *Client) {
	select {
	case h.unregister <- c:
	case <-h.done:
	}
}

// Send hands a message from a browser to the hub.
func (h *Hub) Send(c *Client, msg Message) {
	select {
	case h.incoming <- incoming{from: c, msg: msg}:
	case <-h.done:
	}
}
