package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/rexlx/records/source/definitions"
	"golang.org/x/net/html"
)

func SaveRecordToZinc(record definitions.ZincRecordV2, logger *log.Logger) {
	out, err := json.Marshal(record)
	if err != nil {
		logger.Println(err)
		return
	}
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, ZincUri, bytes.NewBuffer([]byte(out)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Authorization", "Basic "+zincAuth("admin", os.Getenv("ZPWD")))
	if err != nil {
		logger.Println(err)
		return
	}
	res, err := client.Do(req)
	if err != nil {
		logger.Println(err, "hittt")
		return
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		logger.Println("got an unexpected status code", res.StatusCode)
		return
	}
}

func zincAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

const (
	ErcotRTSC              = "https://www.ercot.com/content/cdr/html/real_time_system_conditions.html"
	ErcotSPP               = "https://www.ercot.com/content/cdr/html/real_time_spp.html"
	WeatherUri             = "http://api.weatherapi.com/v1/current.json?key=8c91f83b26994b7b9b8175435222411&q=%v"
	ZincUri                = "http://127.0.0.1:4080/api/_bulkv2"
	CurrentFrequency       = 0
	InstantaneousTimeError = 1
	BAALExceedances        = 2
	ActualDemand           = 3
	Capacity               = 4
	Wind                   = 5
	PVGR                   = 6
	Inertia                = 7
	DC_E                   = 8
	DC_L                   = 9
	DC_N                   = 10
	DC_R                   = 11
	DC_S                   = 12
)

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
	Error                  bool    `json:"error,omitempty"`
	Info                   string  `json:"info"`
	Freq                   float32 `json:"freq,omitempty"`
	InstantaneousTimeError float32 `json:"instantaneous_time_error"`
	BAALExceedances        float32 `json:"baal_exceedances"`
	Demand                 float32 `json:"demand,omitempty"`
	Cap                    float32 `json:"cap,omitempty"`
	WindOutput             float32 `json:"wind_output,omitempty"`
	PVGR                   float32 `json:"pvgr"`
	Inertia                float32 `json:"inertia,omitempty"`
	DC_E                   float32 `json:"dc_e"`
	DC_L                   float32 `json:"dc_l"`
	DC_N                   float32 `json:"dc_n"`
	DC_R                   float32 `json:"dc_r"`
	DC_S                   float32 `json:"dc_s"`
}
type Pair[T, U any] struct {
	Key   T
	Value U
}

func zip[T, U any](ts []T, us []U) []Pair[T, U] {
	if len(ts) != len(us) {
		log.Println("bad lengths")
		return nil
	}
	pairs := make([]Pair[T, U], len(ts))
	for i := 0; i < len(ts); i++ {
		pairs[i] = Pair[T, U]{ts[i], us[i]}
	}
	return pairs
}

func PowerParser(doc *html.Node) []Pair[string, float32] {
	keys := []string{}
	vals := []float32{}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "td" {
			if n.Attr[0].Val == "tdLeft" {
				keys = append(keys, n.FirstChild.Data)
			} else if n.Attr[0].Val == "labelClassCenter" {
				x, err := strconv.ParseFloat(n.FirstChild.Data, 32)
				if err != nil {
					log.Println(err)
				}
				vals = append(vals, float32(x))
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	return zip(keys, vals)
}

func SppParser(doc *html.Node) [][]string {
	keys := []string{}
	vals := []string{}
	// i := 0
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "th" {
			keys = append(keys, n.FirstChild.Data)
			// priceMap[n.FirstChild.Data] = i
			// i++
		} else if n.Type == html.ElementNode && n.Data == "td" {
			vals = append(vals, n.FirstChild.Data)
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	return parseSppVals(vals, len(keys))
}

func parseSppVals(slice []string, step int) [][]string {
	// var s *Spp
	var table [][]string
	for i := 0; i < len(slice); i += step {
		end := i + step
		if end > len(slice) {
			end = len(slice)
		}
		table = append(table, slice[i:end])
	}
	return table
}

func toFloat32(s string) float32 {
	res, err := strconv.ParseFloat(s, 32)
	if err != nil {
		log.Println(err)
	}
	return float32(res)
}
