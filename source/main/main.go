package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/rexlx/records/source/definitions"
	"github.com/rexlx/records/source/services"
)

type Application struct {
	InfoLog         *log.Logger
	ErrorLog        *log.Logger
	Config          *RuntimeConfig
	ApiKey          string
	ServiceRegistry map[string]string
	StateMap        map[string]*serviceDetails
	Mtx             sync.RWMutex
}

type RuntimeConfig struct {
	ZincUri   string                 `json:"zinc_uri"`
	LogPath   string                 `json:"logpath"`
	DataDir   string                 `json:"data_dir"`
	Port      int                    `json:"api_port"`
	Services  []*serviceDetails      `json:"services"`
	WorkerMap *definitions.WorkerMap `json:"-"`
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
	state := make(map[string]*serviceDetails)
	serviceRegistry := make(map[string]string)

	app := Application{
		Config:          &config,
		ServiceRegistry: serviceRegistry,
		InfoLog:         infoLog,
		ErrorLog:        errorLog,
		StateMap:        state,
		Mtx:             sync.RWMutex{},
	}
	AppReceiver(&app)
	app.startServcies(definitions.WorkerMap{
		"weather_monitor": services.GetWeather,
		"rtsc_monitor":    services.GetRealTimeSysCon,
		"spp_monitor":     services.GetSPP,
		"cpu_monitor":     services.CpuMon,
	})
	app.startApi()

}

func (app *Application) startApi() error {
	app.InfoLog.Printf("starting api on port %v", app.Config.Port)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", app.Config.Port),
		Handler: app.apiRoutes(),
	}
	app.createApiKey()
	return srv.ListenAndServe()
}

func (app *Application) startServcies(svs definitions.WorkerMap) {
	app.Config.WorkerMap = &svs
	for _, i := range app.Config.Services {
		i.InfoLog = app.InfoLog
		i.ErrorLog = app.ErrorLog
		i.Store = &definitions.Store{}
		i.Kill = make(chan interface{})
		if _, ok := svs[i.Name]; ok {
			go i.Run(svs[i.Name])
		}
	}
}
