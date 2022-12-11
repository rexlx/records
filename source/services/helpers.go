package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
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
	ErcotRTSC              = "https://www.ercot.com/content/cdr/html/real_time_system_conditions.html"
	ErcotSPP               = "https://www.ercot.com/content/cdr/html/real_time_spp.html"
	WeatherUri             = "http://api.weatherapi.com/v1/current.json?key=&q=%v"
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

type Usage struct {
	Total int
	Idle  int
}

type CpuValue struct {
	Name  string
	Time  time.Time
	Usage float64
}

// GetCpuValues reads the proc stat file, waits for the refresh
// interval and returns the list of values
func GetCpuValues(c chan []*CpuValue, refresh int) {
	now := time.Now()
	values := []*CpuValue{}
	initialPoll, err := pollCpu()
	keys := make([]string, 0, len(initialPoll))
	for k := range initialPoll {
		keys = append(keys, k)
		sort.Strings(keys)
	}
	if err != nil {
		log.Println(err)
	}
	time.Sleep(time.Duration(refresh) * time.Second)
	poll, err := pollCpu()
	if err != nil {
		log.Println(err)
	}
	for _, key := range keys {
		idle := poll[key].Idle - initialPoll[key].Idle
		total := poll[key].Total - initialPoll[key].Total
		values = append(values, &CpuValue{
			Name:  key,
			Usage: 100 * (float64(total) - float64(idle)) / float64(total),
			Time:  now})
	}
	c <- values
}

// pollCpu reads the /proc/stat file and calculates cpu utilization percentage.
// returns a map of cpu stats where: map[cpuN] = n%
func pollCpu() (map[string]*Usage, error) {
	usage := make(map[string]*Usage)
	result := &Usage{}
	contents, err := os.ReadFile("/proc/stat")
	if err != nil {
		return usage, err
	}
	lines := strings.Split(string(contents), "\n")

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 1 {
			continue
		}
		if strings.Contains(fields[0], "cpu") {
			nFields := len(fields)
			for i := 1; i < nFields; i++ {
				if i == 4 {
					val, err := strconv.Atoi(fields[i])
					if err != nil {
						return usage, err
					}
					result.Total += val
					result.Idle += val
				} else {
					val, err := strconv.Atoi(fields[i])
					if err != nil {
						return usage, err
					}
					result.Total += val
				}
				usage[fields[0]] = result
			}
		}
	}
	return usage, nil
}
