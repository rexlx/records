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

// shared application data
type Application struct {
	InfoLog         *log.Logger
	ErrorLog        *log.Logger
	Config          *RuntimeConfig
	ApiKey          string
	ServiceRegistry map[string]string
	StateMap        map[string]*serviceDetails
	Mtx             sync.RWMutex
}

// configuration specific to this runtime
type RuntimeConfig struct {
	ZincUri   string                 `json:"zinc_uri"`
	LogPath   string                 `json:"logpath"`
	DataDir   string                 `json:"data_dir"`
	Port      int                    `json:"api_port"`
	Services  []*serviceDetails      `json:"services"`
	WorkerMap *definitions.WorkerMap `json:"-"`
}

func main() {
	// init an empty runtime config
	var config RuntimeConfig

	// read in the config
	contents, err := os.ReadFile(os.Getenv("RECORDS_CONFIG"))
	if err != nil {
		// if it exists :)
		log.Println("did you set your RECORDS_CONFIG environment variable?")
		log.Fatalln(err)
	}

	// unmarshal into our empty config
	err = json.Unmarshal(contents, &config)
	if err != nil {
		log.Fatalln(err)
	}

	// right now all log streams write to this `config.LogPath`
	file, err := os.OpenFile(config.LogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalln(err)
	}

	// initialize a few things our app will need
	infoLog := log.New(file, "info  ", log.Ldate|log.Ltime)
	errorLog := log.New(file, "error ", log.Ldate|log.Ltime)
	state := make(map[string]*serviceDetails)
	serviceRegistry := make(map[string]string)
	// init the new configured app
	app := Application{
		Config:          &config,
		ServiceRegistry: serviceRegistry,
		InfoLog:         infoLog,
		ErrorLog:        errorLog,
		StateMap:        state,
		Mtx:             sync.RWMutex{},
	}
	// this is how we pass the instance of this application to the scheduler
	AppReceiver(&app)
	// this is where we define our service to function map...for now
	app.startServcies(definitions.WorkerMap{
		"weather_monitor": services.GetWeather,
		"rtsc_monitor":    services.GetRealTimeSysCon,
		"spp_monitor":     services.GetSPP,
		"cpu_monitor":     services.CpuMon,
	})
	// start the api and listen
	app.startApi()

}

// startApi starts an http server
func (app *Application) startApi() error {
	app.InfoLog.Printf("starting api on port %v", app.Config.Port)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", app.Config.Port),
		Handler: app.apiRoutes(),
	}
	app.createApiKey()
	return srv.ListenAndServe()
}

// startServices loops over the workerMap and starts the processes in the background,
// registering it to the application in the process.
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
