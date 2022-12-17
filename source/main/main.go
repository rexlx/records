package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/rexlx/records/source/services"
)

type Application struct {
	InfoLog         *log.Logger
	ErrorLog        *log.Logger
	Config          *RuntimeConfig
	ServiceRegistry map[string]string
	Db              map[string]*Store
	Mtx             sync.RWMutex
}

type RuntimeConfig struct {
	ZincUri  string `json:"zinc_uri"`
	LogPath  string `json:"logpath"`
	DataDir  string `json:"data_dir"`
	Port     int    `json:"api_port"`
	Services struct {
		RTSC ServiceDetails `json:"rtsc_monitor"`
		SPP  ServiceDetails `json:"spp_monitor"`
		WM   ServiceDetails `json:"wapi_monitor"`
		RCPU ServiceDetails `json:"cpu_monitor"`
	} `json:"services"`
}

func main() {
	var config RuntimeConfig

	contents, err := os.ReadFile(os.Getenv("RECORDS_CONFIG"))
	if err != nil {
		log.Println("did you set your RECORDS_CONFIG environment variable?")
		log.Fatalln(err)
	}

	// if the file exists, unmarshal into our empty config
	err = json.Unmarshal(contents, &config)
	if err != nil {
		log.Fatalln(err)
	}

	// right now all log streams write to this file
	file, err := os.OpenFile(config.LogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalln(err)
	}

	infoLog := log.New(file, "info  ", log.Ldate|log.Ltime)
	errorLog := log.New(file, "error ", log.Ldate|log.Ltime)
	database := make(map[string]*Store)
	serviceRegistry := make(map[string]string)

	app := Application{
		Config:          &config,
		ServiceRegistry: serviceRegistry,
		InfoLog:         infoLog,
		ErrorLog:        errorLog,
		Db:              database,
		Mtx:             sync.RWMutex{},
	}
	AppReceiver(&app)
	app.startServcies()
	app.startApi()
	// this block just keeps the program alive for now
	for {
		serviceList := app.getAllServiceData()
		if len(serviceList) > 0 {
			app.InfoLog.Println("performing service health check")
			for k, v := range app.ServiceRegistry {
				if _, ok := app.Db[v]; ok {
					app.InfoLog.Printf("%v (%v) is running. store is: %v", k, v, len(app.Db[v].Records))
				} else {
					app.InfoLog.Printf("this service should be dead.. %v", k)
					delete(app.ServiceRegistry, k)
				}
			}
			time.Sleep(1800 * time.Second)
		} else {
			time.Sleep(1 * time.Second)

		}
	}

}

func (app *Application) startApi() error {
	app.InfoLog.Printf("starting api on port %v", 0)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", app.Config.Port),
		Handler: app.apiRoutes(),
	}
	return srv.ListenAndServe()
}

func (app *Application) startServcies() {
	// set up real time system condition monitor
	app.Config.Services.RTSC.InfoLog = app.InfoLog
	app.Config.Services.RTSC.ErrorLog = app.ErrorLog
	app.Config.Services.RTSC.Store = &Store{}
	app.Config.Services.RTSC.Kill = make(chan interface{})
	go app.Config.Services.RTSC.Run(services.GetRealTimeSysCon)

	// set up settlment point price monitor
	app.Config.Services.SPP.InfoLog = app.InfoLog
	app.Config.Services.SPP.ErrorLog = app.ErrorLog
	app.Config.Services.SPP.Store = &Store{}
	app.Config.Services.SPP.Kill = make(chan interface{})
	go app.Config.Services.SPP.Run(services.GetSPP)

	// set up weather monitor
	app.Config.Services.WM.InfoLog = app.InfoLog
	app.Config.Services.WM.ErrorLog = app.ErrorLog
	app.Config.Services.WM.Store = &Store{}
	app.Config.Services.WM.Kill = make(chan interface{})
	go app.Config.Services.WM.Run(services.GetWeather)

	// cpu mon
	app.Config.Services.RCPU.InfoLog = app.InfoLog
	app.Config.Services.RCPU.ErrorLog = app.ErrorLog
	app.Config.Services.RCPU.Store = &Store{}
	app.Config.Services.RCPU.Kill = make(chan interface{})
	go app.Config.Services.RCPU.Run(services.CpuMon)
}
