package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/rexlx/records/source/definitions"
	"golang.org/x/net/html"
)

func SaveRecordToZinc(zuri string, record definitions.ZincRecordV2, logger *log.Logger) {
	record.Index = fmt.Sprintf("%v-%v", time.Now().Format("200601"), record.Index)
	out, err := json.Marshal(record)
	if err != nil {
		logger.Println(err)
		return
	}
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, zuri, bytes.NewBuffer([]byte(out)))
	req.Header.Set("Content-Type", "application/json")
	// ideally we'd be storing secrets in a secrets manager, this is for dev purposes
	req.Header.Add("Authorization", "Basic "+zincAuth("admin", os.Getenv("ZINC_API_PWD")))
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
	CurrentFrequency = iota
	InstantaneousTimeError
	BAALExceedances
	ActualDemand
	Capacity
	Wind
	PVGR
	Inertia
	DC_E
	DC_L
	DC_N
	DC_R
	DC_S
)

const (
	ErcotRTSC  = "https://www.ercot.com/content/cdr/html/real_time_system_conditions.html"
	ErcotSPP   = "https://www.ercot.com/content/cdr/html/real_time_spp.html"
	WeatherUri = "http://api.weatherapi.com/v1/current.json?key=&q=%v"
	ZincUri    = "http://127.0.0.1:4080/api/_bulkv2"
)

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
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "th" {
			keys = append(keys, n.FirstChild.Data)
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
