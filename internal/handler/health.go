package handler

import "net/http"

func (app *Application) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]string{
		"status":  "ok",
		"version": app.version,
	}
	app.jsonResponse(w, r, http.StatusOK, data)

}
