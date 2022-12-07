package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rexlx/records/source/definitions"
	"github.com/rexlx/records/source/services"
)

func (app *Application) registerService(name, uid string, store Store) {
	app.Mtx.Lock()
	defer app.Mtx.Unlock()
	app.Db[uid] = &store
	app.ServiceRegistry[SanitizeServiceName(name)] = uid
	app.InfoLog.Printf("service registered: %v\t(%v)", uid, name)
}

func (app *Application) removeService(uid string) {
	app.Mtx.Lock()
	defer app.Mtx.Unlock()
	delete(app.Db, uid)
	for k, v := range app.ServiceRegistry {
		if uid == v {
			delete(app.ServiceRegistry, k)
		}
	}
	app.InfoLog.Printf("service removed: %v", uid)
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

func (app *Application) getServiceDataById(uid string) (*Store, error) {
	if _, ok := app.Db[uid]; ok {
		return app.Db[uid], nil
	}
	return &Store{}, fmt.Errorf("no data store linked to that id")
}

func serviceValidator(s *ServiceDetails) error {
	if s.Runtime < 1 || s.Refresh < 1 {
		return fmt.Errorf("wont start service: %v. runtime or refresh set to zero in config", s.Name)
	}
	return nil
}

func (app *Application) handleStore(uid string, store *Store) {
	if len(store.Records) > 0 {
		services.SaveRecordToZinc(app.Config.ZincUri, *store.Records[len(store.Records)-1], app.ErrorLog)
		if len(store.Records) > 99 {
			var emptyStore []*definitions.ZincRecordV2
			app.saveStore("", store)
			store.Records = emptyStore
		}
	}
	// this needs a sync rw mutex i bet
	app.Db[uid] = store
}

func (app *Application) saveStore(uid string, store *Store) {
	now := time.Now().Format("2006-01-02_1504")
	err := app.handleServiceStorageDir(uid)
	if err != nil {
		app.ErrorLog.Println(err)
	}
	outfile := fmt.Sprintf("%v/%v/data_%v.json", app.Config.DataDir, uid, now)

	file, err := os.OpenFile(outfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
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

func (app *Application) handleServiceStorageDir(uid string) error {
	dasPath := filepath.Join(app.Config.DataDir, uid)
	err := os.MkdirAll(dasPath, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}
