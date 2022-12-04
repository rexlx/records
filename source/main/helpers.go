package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rexlx/records/source/definitions"
	"github.com/rexlx/records/source/services"
)

func (app *Application) registerService(uid string, store Store) {
	app.Mtx.Lock()
	defer app.Mtx.Unlock()
	app.Db[uid] = &store
}

func (app *Application) removeService(uid string) {
	app.Mtx.Lock()
	defer app.Mtx.Unlock()
	delete(app.Db, uid)
}

func (app *Application) getAllServiceData() []Store {
	var svs []Store
	app.Mtx.RLock()
	defer app.Mtx.RUnlock()
	for _, store := range app.Db {
		svs = append(svs, *store)
	}
	return svs
}

func (app *Application) getServiceDataById(uid string) *Store {
	return app.Db[uid]
}

func serviceValidator(s *ServiceDetails) error {
	if s.Runtime < 1 || s.Refresh < 1 {
		return fmt.Errorf("wont start service: %v. runtime or refresh set to zero in config", s.Name)
	}
	return nil
}

func (app *Application) handleStore(uid string, store *Store) {
	if len(store.Records) > 0 {
		services.SaveRecordToZinc(*store.Records[len(store.Records)-1], app.ErrorLog)
		if len(store.Records) > 99 {
			var emptyStore []*definitions.ZincRecordV2
			app.saveStore(store)
			store.Records = emptyStore
		}
	}

	app.Db[uid] = store
}

func (app *Application) saveStore(store *Store) {
	now := time.Now().Format("2006-01-02_1504")
	file, err := os.OpenFile(fmt.Sprintf("/home/link/data_%v.json", now), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		app.ErrorLog.Println("couldnt save the store", err)
		return
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	enc.Encode(store.Records)
}

func SanitizeServiceName(name string) string {
	return strings.ReplaceAll(name, " ", "_")
}
