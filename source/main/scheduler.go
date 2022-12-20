package main

import (
	"time"

	"github.com/google/uuid"
	"github.com/rexlx/records/source/definitions"
)

var app *Application

type serviceDetails definitions.ServiceDetails

// Appreceiver is how the scheduler gets access to app wide data
func AppReceiver(a *Application) {
	app = a
}

func (s *serviceDetails) Run(wkr func(c chan definitions.ZincRecordV2)) {
	newStream := make(chan definitions.ZincRecordV2)
	if err := serviceValidator(s); err != nil {
		s.ErrorLog.Println(err)
		return
	}

	uid := uuid.Must(uuid.NewRandom()).String()
	app.registerService(uid, s)
	s.ServiceId = uid
	t, z := s.StartAt[0], s.StartAt[1]
	tz, _ := time.LoadLocation(z)

	// starts immediately
	if !s.Scheduled {
	runtime:
		for {
			s.InfoLog.Printf("%v (%v) is starting. running for %vs every %vs", s.ServiceId, s.Name, s.Runtime, s.Refresh)
			for start := time.Now(); time.Since(start) < time.Second*time.Duration(s.Runtime); {
				select {
				case <-s.Kill:
					break runtime
				default:
				}
				go wkr(newStream)
				msg := <-newStream
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
	schedule:
		for {
			// this branch waits for scheduled time to occur
			if time.Now().In(tz).Format("15:04") != t {
				time.Sleep(1 * time.Second)
			} else {
				s.InfoLog.Printf("%v is starting. running for %vs every %vs", s.Name, s.Runtime, s.Refresh)
				for start := time.Now(); time.Since(start) < time.Second*time.Duration(s.Runtime); {
					select {
					case <-s.Kill:
						break schedule
					default:
					}
					go wkr(newStream)
					msg := <-newStream
					s.Store.Records = append(s.Store.Records, &msg)
					go app.handleStore(s.ServiceId, s.Store)
					time.Sleep(time.Duration(s.Refresh) * time.Second)
				}
				if !s.ReRun {
					s.InfoLog.Println("terminating", s.Name)
					app.removeService(s.ServiceId)
					return
				}
				s.InfoLog.Println(s.Name, "rotating service")
			}
		}
	}
	s.InfoLog.Printf("exit condition for %v (%v) reached.", s.Name, s.ServiceId)
	app.removeService(s.ServiceId)

}
