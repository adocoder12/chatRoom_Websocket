package chat

import (
	"errors"
	"strings"
)

// Message is the JSON that travels both ways:
// browser -> server (join / leave / message) and server -> browser.
type Message struct {
	Type     string `json:"type"`               // "join", "leave", "message", "system"
	Room     string `json:"room,omitempty"`     // target room name
	Username string `json:"username,omitempty"` // set by the server, never trusted from the browser
	Content  string `json:"content,omitempty"`  // message body (or the system text for join/leave)
}

const (
	TypeJoin    = "join"
	TypeLeave   = "leave"
	TypeMessage = "message"
	TypeSystem  = "system" // server -> browser only (welcome, errors)
)

const (
	maxRoomLength    = 30
	maxContentLength = 400 // must fit inside maxMessageSize together with the JSON around it
)

// validate checks a message that came FROM the browser.
// It also trims spaces, so it needs a pointer receiver to change m.
func (m *Message) validate() error {
	m.Room = strings.TrimSpace(m.Room)
	m.Content = strings.TrimSpace(m.Content)

	switch m.Type {
	case TypeJoin:
		if m.Room == "" || len(m.Room) > maxRoomLength {
			return errors.New("room name is required (max 30 characters)")
		}
	case TypeLeave:
		// nothing to check: the hub knows which room the client is in
	case TypeMessage:
		if m.Content == "" || len(m.Content) > maxContentLength {
			return errors.New("message is empty or too long")
		}
	default:
		return errors.New("unknown message type")
	}
	return nil
}
