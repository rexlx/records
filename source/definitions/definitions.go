package definitions

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
