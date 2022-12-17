package main

import (
	"fmt"
	"net/http"

	"github.com/rexlx/records/source/definitions"
)

type jsonResponse definitions.JsonResponse
type box map[string]interface{}

func (app *Application) ListServices(w http.ResponseWriter, r *http.Request) {
	_ = app.writeJSON(w, http.StatusOK, app.ServiceRegistry)
}

func (app *Application) KillService(w http.ResponseWriter, r *http.Request) {
	type service struct {
		Id string `json:"id"`
	}
	var sid service
	var data jsonResponse
	err := app.readJSON(w, r, &sid)
	if err != nil {
		app.ErrorLog.Println(err)
		data.Error = true
		data.Message = "invalid json"
		_ = app.writeJSON(w, http.StatusBadRequest, data)
	}
	app.InfoLog.Println("KillService called on", sid.Id)
	close(app.StateMap[sid.Id].Kill)
	msg := jsonResponse{
		Error:   false,
		Message: fmt.Sprintf("kill signal sent to %v", sid.Id),
	}
	_ = app.writeJSON(w, http.StatusOK, msg)
}
