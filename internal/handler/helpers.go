package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// envelope is the top-level shape of every JSON response.
type envelope struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

func (app *Application) writeJSON(w http.ResponseWriter, status int, data any) error {
	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("encode response: %w", err)
	}
	body = append(body, '\n')

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(body); err != nil {
		app.logger.Warn("write response", "err", err)
	}
	return nil
}

// jsonResponse sends {"data": data} with the given status.
func (app *Application) jsonResponse(w http.ResponseWriter, r *http.Request, status int, data any) {
	if err := app.writeJSON(w, status, envelope{Data: data}); err != nil {
		app.serverError(w, r, err)
	}
}

// errorResponse sends {"error": msg} with the given status.
func (app *Application) errorResponse(w http.ResponseWriter, r *http.Request, status int, msg string) {
	if err := app.writeJSON(w, status, envelope{Error: msg}); err != nil {
		app.logger.Error("write error response",
			"err", err,
			"method", r.Method,
			"uri", r.URL.RequestURI(),
		)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// serverError logs the real error and sends a generic 500 to the client.
func (app *Application) serverError(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Error("server error",
		"err", err,
		"method", r.Method,
		"uri", r.URL.RequestURI(),
	)
	app.errorResponse(w, r, http.StatusInternalServerError, "the server encountered a problem")
}

// notFound sends a 404.
func (app *Application) notFound(w http.ResponseWriter, r *http.Request) {
	app.logger.Warn("not found",
		"method", r.Method,
		"uri", r.URL.RequestURI(),
	)
	app.errorResponse(w, r, http.StatusNotFound, "not found")
}

// methodNotAllowed sends a 405.
func (app *Application) methodNotAllowed(w http.ResponseWriter, r *http.Request) {
	app.errorResponse(w, r, http.StatusMethodNotAllowed, "method not allowed")
}

// badRequest logs the problem and sends a 400 with the error message.
// Always pass a non-nil error.
func (app *Application) badRequest(w http.ResponseWriter, r *http.Request, msg string, err error) {
	app.logger.Warn("bad request",
		"msg", msg,
		"err", err,
		"method", r.Method,
		"uri", r.URL.RequestURI(),
	)
	app.errorResponse(w, r, http.StatusBadRequest, msg)
}
