package handler

import (
	"errors"
	"net/http"

	"github.com/adocoder12/chatRoom/internal/chat"
	"github.com/adocoder12/chatRoom/static"
	"github.com/gorilla/websocket"
)

// No CheckOrigin set: gorilla's default only accepts requests whose Origin
// matches the Host (same-origin). That's safe while index.html is served by
// this same server. When another origin needs access (e.g. a separate
// frontend), add ALLOWED_ORIGIN to config and check it here. Never "allow all".
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (app *Application) HomeHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFileFS(w, r, static.Files, "template/html/index.html")
}

func (app *Application) ChatHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" || len(username) > 30 {
		app.badRequest(w, r, "username is required (max 30 characters)", errors.New("invalid username"))
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// Upgrade already wrote the error response. Only log.
		app.logger.Error("upgrade failed", "err", err)
		return
	}
	// No defer conn.Close() here: this handler returns right after Start(),
	// and a defer would close the connection immediately. The pumps own it now.

	app.logger.Info("client connected", "username", username, "addr", r.RemoteAddr)

	client := chat.NewClient(app.hub, conn, username, app.logger)
	app.hub.Register(client) // register first, so the client gets messages as soon as it starts
	client.Start()
}
