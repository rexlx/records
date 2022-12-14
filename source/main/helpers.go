package main

import (
	"bytes"
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
	// also remove from registry, this may change in the future idk
	for k, v := range app.ServiceRegistry {
		if uid == v {
			delete(app.ServiceRegistry, k)
		}
	}
}

// getAllServiceData returns an unordered list of service state maps
func (app *Application) getAllServiceData() []*serviceDetails {
	var svs []*serviceDetails
	app.Mtx.RLock()
	defer app.Mtx.RUnlock()
	for _, svc := range app.StateMap {
		svs = append(svs, svc)
	}
	return svs
}

// getStore returns a list of records if they exist or an error
func (app *Application) getStore(uid string) ([]*definitions.ZincRecordV2, error) {
	if _, ok := app.StateMap[uid]; ok {
		return app.StateMap[uid].Store.Records, nil
	}
	return []*definitions.ZincRecordV2{}, errors.New("invalid id")
}

// getServiceDataById returns the state of a specific service
func (app *Application) getServiceDataById(uid string) (*serviceDetails, error) {
	if _, ok := app.StateMap[uid]; ok {
		return app.StateMap[uid], nil
	}
	return &serviceDetails{}, fmt.Errorf("no data store linked to that id")
}

// getAllServiceCounters returns a list of all counters premarshalled into bytes
func (app *Application) getAllServiceCounters() []byte {
	type statContainer struct {
		Name     string                `json:"name"`
		Counters *definitions.Counters `json:"counters"`
	}

	var stats []*statContainer
	for _, svc := range app.StateMap {
		s := &statContainer{
			Name:     svc.Name,
			Counters: svc.Store.Counters,
		}
		stats = append(stats, s)
	}
	out, err := json.Marshal(stats)
	if err != nil {
		app.ErrorLog.Println(err)
		return []byte{}
	}
	return out
}

// getLoadedServices returns a list of services premarshalled into bytes
func (app *Application) getLoadedServices() []byte {
	out, err := json.Marshal(app.Config.Services)
	if err != nil {
		app.InfoLog.Println(err)
	}
	return out
}

// getDefaults effectively (or perhaps rather, is intended to) sets a service details
// to that of a matching service loaded into the service list
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

// nameApplication fetches an adjective-noun style random name from an api and sets
// the apps ID to that (and thus, the key file name in the bucket)
func (app *Application) nameApplication() {
	url := `https://namer.nullferatu.com`
	var pl struct {
		Data string `json:"data"`
	}
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		app.ErrorLog.Println(err)
		return
	}

	res, err := client.Do(req)
	if err != nil {
		app.ErrorLog.Println(err)
		return
	}

	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		app.ErrorLog.Println(err)
		return
	}
	err = json.Unmarshal(data, &pl)
	if err != nil {
		app.ErrorLog.Println(err)
		return
	}
	app.Id = pl.Data
}

// handleStore sends the records to be indexed (look into zinclabs).
func (app *Application) handleStore(uid string, store *definitions.Store) {
	if len(store.Records) == app.StateMap[uid].Store.Counters.Signature {
		err := errors.New("service progressed, but state was unchanged")
		app.ErrorLog.Println(err, uid)
		app.StateMap[uid].Store.Errors = append(app.StateMap[uid].Store.Errors, &err)
		return
	}
	if len(store.Records) > app.StateMap[uid].Store.Counters.Signature {
		app.StateMap[uid].Store.Counters.Signature = len(store.Records)
		services.SaveRecordToZinc(app.Config.ZincUri, *store.Records[len(store.Records)-1], app.ErrorLog)
	}
	if len(store.Records) > 199 {
		var emptyStore []*definitions.ZincRecordV2
		// saveStore slated for removal
		// app.saveStore(uid, store)
		app.StateMap[uid].Store.Counters.StoreEmptied += 1
		app.StateMap[uid].Store.Counters.Signature = 0
		store.Records = emptyStore
	}
	// this needs a sync rw mutex i bet
	app.StateMap[uid].Store = store
}

// saves slice to disk. this was for initial testing, slated for removal
func (app *Application) saveStore(uid string, store *definitions.Store) {
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

// SanitizeServiceName replaces white space with underscores
func SanitizeServiceName(name string) string {
	return strings.ReplaceAll(name, " ", "_")
}

// handleServiceStorageDir creates a directory if it doesnt exist.
func (app *Application) handleServiceStorageDir(uid string) error {
	dasPath := filepath.Join(app.Config.DataDir, uid)
	err := os.MkdirAll(dasPath, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

// errorJSON writes an error message to the responseWriter
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

// writeJSON writes a message to the responseWriter
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

// readJSON reads and decodes the body of the http request
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

// createApiKey creates a 40 character api key
func (app *Application) createApiKey() {
	// generate our random 40 ch hex string
	val, err := genRandomString()
	if err != nil {
		app.ErrorLog.Println(err)
		return
	}
	// create our hash, but use a min cost of 12 instead of 10
	key, err := bcrypt.GenerateFromPassword([]byte(val), 12)
	if err != nil {
		app.ErrorLog.Println(err)
		return
	}
	// that is our api key
	app.ApiKey = string(key)
	// and no one shall know but us
	app.PlaceKey(&val, &app.Id)
}

// PlaceKey writes an api key to a file given that file name
func (app *Application) PlaceKey(key, name *string) {
	type payload struct {
		Key  string `json:"key"`
		Name string `json:"name"`
	}
	var pl payload
	pl.Key = *key
	pl.Name = *name
	out, err := json.Marshal(pl)
	if err != nil {
		app.ErrorLog.Println(err)
		return
	}
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, os.Getenv("KEY_STORE"), bytes.NewBuffer([]byte(out)))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		app.ErrorLog.Println(err)
		return
	}
	res, err := client.Do(req)
	if err != nil {
		app.ErrorLog.Println("http client failure", err)
		return
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		app.ErrorLog.Println("got an unexpected status code", res.StatusCode)
		return
	}
}

// validateKey compares the Bearer token against the hash stored as the key
// it is also the middleware used to authenticate anything in the `/app` route.
// this is expensive around ~350ms
func (app *Application) validateKey(r *http.Request) (bool, error) {
	app.InfoLog.Printf("ACCESS : validating request for %v", r.RemoteAddr)
	header := r.Header.Get("Authorization")
	if header == "" {
		return false, errors.New("no auth headers")
	}

	values := strings.Split(header, " ")
	if len(values) != 2 || values[0] != "Bearer" {
		return false, errors.New("bad auth headers")
	}

	// ensuring the len is what we expect is an unnecessary step but also a very quick way
	// to deter unwanted access before we compare the hash, the most cpu intensive part of this
	if len(values[1]) != 40 {
		return false, errors.New("that token isn't tokeny enough")
	}
	// this is what costs us so much
	err := bcrypt.CompareHashAndPassword([]byte(app.ApiKey), []byte(values[1]))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			// invalid password
			app.InfoLog.Printf("ACCESS : failed authentication %v", r.RemoteAddr)
			return false, errors.New("-")
		default:
			return false, err
		}
	}
	app.InfoLog.Printf("ACCESS : successful authentication %v", r.RemoteAddr)
	return true, nil
}

// genRandomString creates the 40 ch hex string needed for the api key
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
