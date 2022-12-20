package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/rexlx/records/source/definitions"
	"github.com/rexlx/records/source/services"
	"golang.org/x/crypto/bcrypt"
)

// registerService adds a service to the application state map as well as the service registry
// the service registry is an internal convention and is a map of any service that is or was
// running since the last start
func (app *Application) registerService(uid string, state *serviceDetails) {
	app.Mtx.Lock()
	defer app.Mtx.Unlock()
	app.ServiceRegistry[SanitizeServiceName(state.Name)] = uid
	app.StateMap[uid] = state
}

// removeService removes a services from the application state map
func (app *Application) removeService(uid string) {
	app.Mtx.Lock()
	defer app.Mtx.Unlock()
	delete(app.StateMap, uid)
	// keep services in reg for now
	for k, v := range app.ServiceRegistry {
		if uid == v {
			delete(app.ServiceRegistry, k)
		}
	}
}

// getAllServiceData returns an unordered list of service state maps
// i dont know why i need this.
func (app *Application) getAllServiceData() []*serviceDetails {
	var svs []*serviceDetails
	app.Mtx.RLock()
	defer app.Mtx.RUnlock()
	for _, svc := range app.StateMap {
		svs = append(svs, svc)
	}
	return svs
}

// getServiceDataById returns the state of a specific service
func (app *Application) getServiceDataById(uid string) (*serviceDetails, error) {
	if _, ok := app.StateMap[uid]; ok {
		return app.StateMap[uid], nil
	}
	return &serviceDetails{}, fmt.Errorf("no data store linked to that id")
}

func (app *Application) getLoadedServices() []byte {
	out, err := json.Marshal(app.Config.Services)
	if err != nil {
		app.InfoLog.Println(err)
	}
	return out
}

func (app *Application) getDefaults(s *serviceDetails) {
	for _, i := range app.Config.Services {
		if i.Name == s.Name {
			s.Runtime = i.Runtime
			s.Refresh = i.Refresh
			s.ReRun = i.ReRun
			s.StartAt = i.StartAt
		}
	}
}

// handleStore sends the records to be indexed (look into zinclabs). additionally
// after a given time, it saves its list of records to the specified date dir
func (app *Application) handleStore(uid string, store *definitions.Store) {
	if len(store.Records) > 0 {
		services.SaveRecordToZinc(app.Config.ZincUri, *store.Records[len(store.Records)-1], app.ErrorLog)
		if len(store.Records) > 99 {
			var emptyStore []*definitions.ZincRecordV2
			app.saveStore(uid, store)
			store.Records = emptyStore
		}
	}
	// this needs a sync rw mutex i bet
	app.StateMap[uid].Store = store
}

// saves storage slice to disk
func (app *Application) saveStore(uid string, store *definitions.Store) {
	// now := time.Now().Format("2006-01-02_1504")
	err := app.handleServiceStorageDir(uid)
	if err != nil {
		app.ErrorLog.Println(err)
	}
	outfile := fmt.Sprintf("%v/%v/data.json", app.Config.DataDir, uid)

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

func (app *Application) errorJSON(w http.ResponseWriter, err error, status ...int) error {
	statusCode := http.StatusBadRequest

	if len(status) > 0 {
		statusCode = status[0]
	}

	var payload jsonResponse
	payload.Error = true
	payload.Message = err.Error()

	app.writeJSON(w, statusCode, payload)
	return nil
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
		app.ErrorLog.Println("readJSON encountered a fatal error", err)
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
	app.ApiKey = string(key)
	app.InfoLog.Println("first time admin key:", val)
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
	err := bcrypt.CompareHashAndPassword([]byte(app.ApiKey), []byte(values[1]))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			// invalid password
			return false, errors.New("-")
		default:
			return false, err
		}
	}
	return true, nil
}

func genRandomString() (string, error) {
	// 40 hex chars
	bytes := make([]byte, 20)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// serviceValidator ensures a configured service meets whatever evolving criteria may...evolve
func serviceValidator(s *serviceDetails) error {
	if s.Runtime < 1 || s.Refresh < 1 {
		return fmt.Errorf("wont start service: %v. runtime or refresh set to zero in config", s.Name)
	}
	return nil
}
