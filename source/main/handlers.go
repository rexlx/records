package main

import (
	"fmt"
	"net/http"

	"github.com/rexlx/records/source/definitions"
)

type jsonResponse definitions.JsonResponse

// type box map[string]interface{}

// ListServices lists all running services
func (app *Application) ListServices(w http.ResponseWriter, r *http.Request) {
	_ = app.writeJSON(w, http.StatusOK, app.ServiceRegistry)
}

// ListLoaded lists all services that we initialized during startup
func (app *Application) ListLoaded(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write(app.getLoadedServices())
	if err != nil {
		app.ErrorLog.Println(err)
		return
	}
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

func (app *Application) StartService(w http.ResponseWriter, r *http.Request) {
	type service struct {
		Name string `json:"name"`
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

	if _, ok := app.ServiceRegistry[sid.Name]; !ok {
		newService := serviceDetails{
			Name:     sid.Name,
			InfoLog:  app.InfoLog,
			ErrorLog: app.ErrorLog,
			Store:    &definitions.Store{},
			Kill:     make(chan interface{}),
		}
		app.getDefaults(&newService)

		if _, ok := (*app.Config.WorkerMap)[newService.Name]; ok {
			wkr := (*app.Config.WorkerMap)[newService.Name]
			go newService.Run(wkr)
			msg := jsonResponse{
				Error:   false,
				Message: fmt.Sprintf("started the following service %v", sid.Name),
			}
			_ = app.writeJSON(w, http.StatusOK, msg)
		}
	} else {
		msg := jsonResponse{
			Error:   true,
			Message: fmt.Sprintf("that service is running! %v", sid.Name),
		}
		_ = app.writeJSON(w, http.StatusOK, msg)
	}
}
