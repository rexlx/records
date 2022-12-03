package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/rexlx/records/source/definitions"
	"github.com/rexlx/records/source/services"
)

var app *Application

type Store struct {
	Records []*definitions.ZincRecordV2
}
type ServiceDetails struct {
	Name      string   `json:"name"`
	Index     string   `json:"index"`
	Runtime   int      `json:"runtime"`
	Refresh   int      `json:"refresh"`
	ReRun     bool     `json:"rerun"`
	Scheduled bool     `json:"scheduled"`
	StartAt   []string `json:"start_at"`
	ServiceId string
	Stream    chan definitions.ZincRecordV2
	InfoLog   *log.Logger
	ErrorLog  *log.Logger
	Store     *Store
}

// Appreceiver is how the rpcs gain access to app wide data
func AppReceiver(a *Application) {
	app = a
}

func (s *ServiceDetails) Run(c chan definitions.ZincRecordV2, wkr func(c chan definitions.ZincRecordV2)) {
	if err := serviceValidator(s); err != nil {
		s.ErrorLog.Println(err)
		return
	}
	uid := uuid.Must(uuid.NewRandom()).String()
	app.registerService(uid, *s.Store)
	// defer app.removeService(uid)
	app.InfoLog.Printf("new service registered: %v -> %v", s.Name, uid)
	s.ServiceId = uid
	t, z := s.StartAt[0], s.StartAt[1]
	tz, _ := time.LoadLocation(z)

	// starts immediately
	if !s.Scheduled {
		for {
			s.InfoLog.Printf("%v is starting. running for %vs every %vs", s.Name, s.Runtime, s.Refresh)
			for i := 0; i < (s.Runtime / s.Refresh); i++ {
				go wkr(c)
				msg := <-c
				s.Store.Records = append(s.Store.Records, &msg)
				go app.handleStore(s.ServiceId, s.Store)
				time.Sleep(time.Duration(s.Refresh) * time.Second)
			}
			if !s.ReRun {
				s.InfoLog.Println("terminating", s.Name)
				app.removeService(uid)
				return
			}
			s.InfoLog.Println(s.Name, "rotating service")
		}
	}

	if s.Scheduled {
		s.InfoLog.Printf("%v initialized. waiting for work to start at %v", s.Name, s.StartAt)
		for {
			// this branch waits for scheduled time to occur
			if time.Now().In(tz).Format("15:04") != t {
				time.Sleep(1 * time.Second)
			} else {
				s.InfoLog.Printf("%v is starting. running for %vs every %vs", s.Name, s.Runtime, s.Refresh)
				for i := 0; i < (s.Runtime / s.Refresh); i++ {
					go wkr(c)
					msg := <-c
					s.Store.Records = append(s.Store.Records, &msg)
					go app.handleStore(s.ServiceId, s.Store)
					time.Sleep(time.Duration(s.Refresh) * time.Second)
				}
				if !s.ReRun {
					s.InfoLog.Println("terminating", s.Name)
					app.removeService(uid)
					return
				}
				s.InfoLog.Println(s.Name, "rotating service")
			}
		}
	}

}

func serviceValidator(s *ServiceDetails) error {
	if s.Runtime < 1 || s.Refresh < 1 {
		return fmt.Errorf("wont start service: %v. runtime or refresh set to zero in config", s.Name)
	}
	return nil
}

func (app *Application) handleStore(uid string, store *Store) {
	var emptyStore []*definitions.ZincRecordV2
	services.SaveRecordToZinc(*store.Records[len(store.Records)-1], app.ErrorLog)

	if len(store.Records) > 29 {
		app.saveStore(store)
		store.Records = emptyStore
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
