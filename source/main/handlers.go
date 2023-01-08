package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/rexlx/records/source/definitions"
)

type jsonResponse definitions.JsonResponse
type service struct {
	Id string `json:"id"`
}

// ListServices lists all running services
func (app *Application) ListServices(w http.ResponseWriter, r *http.Request) {
	_ = app.writeJSON(w, http.StatusOK, app.ServiceRegistry)
}

func (app *Application) ListAllCounters(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write(app.getAllServiceCounters())
	if err != nil {
		app.ErrorLog.Println(err)
		return
	}
}

func (app *Application) GetRuntime(w http.ResponseWriter, r *http.Request) {
	var sid service
	var data jsonResponse
	err := app.readJSON(w, r, &sid)
	if err != nil {
		app.ErrorLog.Println(err)
		data.Error = true
		data.Message = "invalid json"
		_ = app.writeJSON(w, http.StatusBadRequest, data)
	}
	if _, ok := app.StateMap[sid.Id]; ok {
		rt := time.Since(app.StateMap[sid.Id].Store.Counters.Start).Minutes()
		msg := jsonResponse{
			Error: false,
			Data:  rt,
		}
		_ = app.writeJSON(w, http.StatusOK, msg)
	} else {
		msg := jsonResponse{
			Error:   true,
			Message: "id does not exist",
		}
		_ = app.writeJSON(w, http.StatusBadRequest, msg)
	}
}

func (app *Application) GetErrorsById(w http.ResponseWriter, r *http.Request) {
	var sid service
	var data jsonResponse
	err := app.readJSON(w, r, &sid)
	if err != nil {
		app.ErrorLog.Println(err)
		data.Error = true
		data.Message = "invalid json"
		_ = app.writeJSON(w, http.StatusBadRequest, data)
	}
	if _, ok := app.StateMap[sid.Id]; ok {
		msg := jsonResponse{
			Error: false,
			Data:  app.StateMap[sid.Id].Store.Errors,
		}
		_ = app.writeJSON(w, http.StatusOK, msg)
	} else {
		msg := jsonResponse{
			Error:   true,
			Message: "id does not exist",
		}
		_ = app.writeJSON(w, http.StatusBadRequest, msg)
	}

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

func (app *Application) GetStore(w http.ResponseWriter, r *http.Request) {
	var sid service
	var data jsonResponse
	err := app.readJSON(w, r, &sid)
	if err != nil {
		app.ErrorLog.Println(err)
		data.Error = true
		data.Message = "invalid json"
		_ = app.writeJSON(w, http.StatusBadRequest, data)
	}
	store, err := app.getStore(sid.Id)
	if err != nil {
		data.Error = true
		data.Message = err.Error()
		_ = app.writeJSON(w, http.StatusBadRequest, data)
	}

	msg := jsonResponse{
		Error:   false,
		Message: fmt.Sprintf("fetched store for %v", sid.Id),
		Data:    store,
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
