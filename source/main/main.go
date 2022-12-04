package main

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"

	"github.com/rexlx/records/source/definitions"
	"github.com/rexlx/records/source/services"
)

type Application struct {
	InfoLog         *log.Logger
	ErrorLog        *log.Logger
	Config          RuntimeConfig
	ServiceRegistry map[string]string
	RtscStream      chan definitions.ZincRecordV2
	SppStream       chan definitions.ZincRecordV2
	WapiStream      chan definitions.ZincRecordV2
	Db              map[string]*Store
	Mtx             sync.RWMutex
}

type RuntimeConfig struct {
	ZincUri  string `json:"zinc_uri"`
	LogPath  string `json:"logpath"`
	Services struct {
		RTSC ServiceDetails `json:"rtsc_monitor"`
		SPP  ServiceDetails `json:"spp_monitor"`
		WM   ServiceDetails `json:"wapi_monitor"`
	} `json:"services"`
}

func main() {
	var config RuntimeConfig

	contents, err := os.ReadFile(os.Getenv("CFG"))
	if err != nil {
		log.Println("did you set your CFG environment variable?")
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
	rtscStream := make(chan definitions.ZincRecordV2)
	sppStream := make(chan definitions.ZincRecordV2)
	wapiStream := make(chan definitions.ZincRecordV2)
	database := make(map[string]*Store)
	serviceRegistry := make(map[string]string)

	app := Application{
		Config:          config,
		ServiceRegistry: serviceRegistry,
		InfoLog:         infoLog,
		ErrorLog:        errorLog,
		RtscStream:      rtscStream,
		SppStream:       sppStream,
		WapiStream:      wapiStream,
		Db:              database,
		Mtx:             sync.RWMutex{},
	}
	AppReceiver(&app)
	app.startServcies()
	for {
		serviceList := app.getServices()
		if len(serviceList) < 1 {
			app.InfoLog.Println("no remaining services are scheduled. waiting")
			// os.Exit(0)
		}
		time.Sleep(60 * time.Second)
	}

}

func (app *Application) startServcies() {
	// set up real time system condition monitor
	app.Config.Services.RTSC.InfoLog = app.InfoLog
	app.Config.Services.RTSC.ErrorLog = app.ErrorLog
	app.Config.Services.RTSC.Stream = app.RtscStream
	app.Config.Services.RTSC.Store = &Store{}
	go app.Config.Services.RTSC.Run(app.RtscStream, services.GetRealTimeSysCon)

	// set up settlment point price monitor
	app.Config.Services.SPP.InfoLog = app.InfoLog
	app.Config.Services.SPP.ErrorLog = app.ErrorLog
	app.Config.Services.SPP.Stream = app.SppStream
	app.Config.Services.SPP.Store = &Store{}
	go app.Config.Services.SPP.Run(app.SppStream, services.GetSPP)

	// set up weather monitor
	app.Config.Services.WM.InfoLog = app.InfoLog
	app.Config.Services.WM.ErrorLog = app.ErrorLog
	app.Config.Services.WM.Stream = app.WapiStream
	app.Config.Services.WM.Store = &Store{}
	go app.Config.Services.WM.Run(app.WapiStream, services.GetWeather)

	app.InfoLog.Println("finished starting services")
}
