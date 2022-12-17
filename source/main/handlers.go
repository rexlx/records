package main

import (
	"encoding/json"
	"net/http"

	"github.com/rexlx/records/source/definitions"
)

type jsonResponse definitions.JsonResponse
type box map[string]interface{}

func (app *Application) ListServices(w http.ResponseWriter, r *http.Request) {
	out, err := json.Marshal(app.ServiceRegistry)
	if err != nil {
		app.ErrorLog.Println(err)
		return
	}
	data := jsonResponse{
		Error:   false,
		Message: "running services",
		Data:    box{"services": out},
	}
	_ = app.writeJSON(w, http.StatusOK, data)
}
