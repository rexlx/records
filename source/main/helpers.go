package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rexlx/records/source/definitions"
	"github.com/rexlx/records/source/services"
	"golang.org/x/crypto/bcrypt"
)

func (app *Application) registerService(name, uid string, state *ServiceDetails) {
	app.Mtx.Lock()
	defer app.Mtx.Unlock()
	app.Db[uid] = state.Store
	app.ServiceRegistry[SanitizeServiceName(name)] = uid
	app.StateMap[uid] = state
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
			app.saveStore(uid, store)
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

func (app *Application) errorJSON(w http.ResponseWriter, err error, status ...int) {
	statusCode := http.StatusBadRequest

	if len(status) > 0 {
		statusCode = status[0]
	}

	var payload jsonResponse
	payload.Error = true
	payload.Message = err.Error()

	app.writeJSON(w, statusCode, payload)
}

func (app *Application) writeJSON(w http.ResponseWriter, status int, data interface{}, headers ...http.Header) error {
	var out []byte
	output, err := json.Marshal(data)
	if err != nil {
		return err
	}
	out = output

	if len(headers) > 0 {
		for key, value := range headers[0] {
			w.Header()[key] = value
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(out)
	if err != nil {
		return err
	}

	return nil
}

func (app *Application) readJSON(w http.ResponseWriter, r *http.Request, data interface{}) error {
	// 5.9MiB
	maxBytes := 6206016
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))
	dec := json.NewDecoder(r.Body)
	//--:REX you changed `err := dec.Decode(data)` -> `err := dec.Decode(&data)`
	err := dec.Decode(data)
	if err != nil {
		app.ErrorLog.Println("BRUHhhhhhhhhh")
		return err
	}

	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("error parsing json")
	}
	return nil
}

func (app *Application) createApiKey() {
	val, err := genRandomString()
	if err != nil {
		app.ErrorLog.Println(err)
		return
	}
	key, err := bcrypt.GenerateFromPassword([]byte(val), 12)
	if err != nil {
		app.ErrorLog.Println(err)
		return
	}
	app.ServiceRegistry["this_api"] = string(key)
}
func (app *Application) validateKey(r *http.Request) (bool, error) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return false, errors.New("no auth headers")
	}

	values := strings.Split(header, " ")
	if len(values) != 2 || values[0] != "Bearer" {
		return false, errors.New("bad auth headers")
	}
	log.Println(values[1])
	err := bcrypt.CompareHashAndPassword([]byte(app.ServiceRegistry["this_api"]), []byte(values[1]))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			// invalid password
			return false, nil
		default:
			return false, err
		}
	}
	return true, nil
}

func genRandomString() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
