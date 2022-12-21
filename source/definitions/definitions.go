package definitions

import (
	"log"
	"time"
)

type ZincRecordV2 struct {
	Index   string                   `json:"index"`
	Records []map[string]interface{} `json:"records"`
}

type WeatherResponse struct {
	Location *Location `json:"location"`
	Current  *Weather  `json:"current"`
}
type Weather struct {
	AsOfEpoch     uint    `json:"last_updated_epoch"`
	AsOf          string  `json:"last_updated"`
	TempC         float64 `json:"temp_c"`
	TempF         float64 `json:"temp_f"`
	FeelslikeC    float64 `json:"feelslike_c"`
	FeelslikeF    float64 `json:"feelslike_f"`
	IsDay         int     `json:"is_day"`
	WindMPH       float64 `json:"wind_mph"`
	WindKPH       float64 `json:"wind_kph"`
	WindDirection string  `json:"wind_dir"`
	WindDegree    float64 `json:"wind_degree"`
	PressureMb    float64 `json:"pressure_mb"`
	PressureIn    float64 `json:"pressure_in"`
	PrecipMM      float64 `json:"precip_mm"`
	PrecipIn      float64 `json:"precip_in"`
	Humidity      int     `json:"humidity"`
	Cloud         int     `json:"cloud"`
	UV            float64 `json:"uv"`
	Condition     struct {
		Text string `json:"text"`
	} `json:"condition"`
}

type Location struct {
	Name      string  `json:"name"`
	Region    string  `json:"region"`
	Country   string  `json:"country"`
	TimeZone  string  `json:"tz_id"`
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
	LocalTime string  `json:"localtime"`
}

type Spp struct {
	Date string
	HbBusAvg,
	HbHouston,
	HbHubAvg,
	HbNorth,
	HbPan,
	HbSouth,
	HbWest,
	LzAen,
	LzCps,
	LzHouston,
	LzLcra,
	LzNorth,
	LzRaybn,
	LzSouth,
	LzWest float32
}
type SysConResponse struct {
	Error                  bool      `json:"error"`
	Info                   string    `json:"info"`
	Time                   time.Time `json:"time"`
	Freq                   float32   `json:"freq"`
	InstantaneousTimeError float32   `json:"instantaneous_time_error"`
	BAALExceedances        float32   `json:"baal_exceedances"`
	Demand                 float32   `json:"demand"`
	AvgNetLoad             float32   `json:"avg_net_load"`
	Cap                    float32   `json:"cap"`
	WindOutput             float32   `json:"wind_output"`
	PVGR                   float32   `json:"pvgr"`
	Inertia                float32   `json:"inertia"`
	DC_E                   float32   `json:"dc_e"`
	DC_L                   float32   `json:"dc_l"`
	DC_N                   float32   `json:"dc_n"`
	DC_R                   float32   `json:"dc_r"`
	DC_S                   float32   `json:"dc_s"`
}

type JsonResponse struct {
	Error   bool        `json:"error"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type Counters struct {
	Start        time.Time
	StoreEmptied int
	Iterations   int
	Signature    int
}
type Store struct {
	Records  []*ZincRecordV2
	Errors   []*error
	Counters *Counters
}

type ServiceDetails struct {
	Name      string            `json:"name"`
	Index     string            `json:"index"`
	Runtime   int               `json:"runtime"`
	Refresh   int               `json:"refresh"`
	ReRun     bool              `json:"rerun"`
	Scheduled bool              `json:"scheduled"`
	StartAt   []string          `json:"start_at"`
	ServiceId string            `json:"id"`
	Waiting   bool              `json:"-"`
	Kill      chan interface{}  `json:"-"`
	Stream    chan ZincRecordV2 `json:"-"`
	InfoLog   *log.Logger       `json:"-"`
	ErrorLog  *log.Logger       `json:"-"`
	Store     *Store            `json:"-"`
}

type WorkerMap map[string]func(chan ZincRecordV2)
