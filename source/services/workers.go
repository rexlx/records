package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rexlx/performance"
	"github.com/rexlx/records/source/definitions"
	"golang.org/x/net/html"
)

func GetRealTimeSysCon(c chan definitions.ZincRecordV2) {
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, ErcotRTSC, nil)
	if err != nil {
		log.Println(err)
		return
	}
	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		log.Println(err)
		return
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		log.Println(err)
		return
	}
	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		log.Println(err)
		return
	}
	result := PowerParser(doc)
	rtsc_res := definitions.SysConResponse{
		Error:                  false,
		Time:                   time.Now(),
		Freq:                   result[CurrentFrequency].Value,
		InstantaneousTimeError: result[InstantaneousTimeError].Value,
		BAALExceedances:        result[BAALExceedances].Value,
		Demand:                 result[ActualDemand].Value,
		AvgNetLoad:             result[AvgNetLoad].Value,
		Cap:                    result[Capacity].Value,
		WindOutput:             result[Wind].Value,
		PVGR:                   result[PVGR].Value,
		Inertia:                result[Inertia].Value,
		DC_E:                   result[DC_E].Value,
		DC_L:                   result[DC_L].Value,
		DC_N:                   result[DC_N].Value,
		DC_R:                   result[DC_R].Value,
		DC_S:                   result[DC_S].Value,
	}
	var envelope []map[string]interface{}
	var tmp map[string]interface{}
	out, err := json.Marshal(rtsc_res)
	if err != nil {
		log.Println(err)
		return
	}
	json.Unmarshal(out, &tmp)
	envelope = append(envelope, tmp)
	c <- definitions.ZincRecordV2{
		Index:   "ercotRTSC",
		Records: envelope,
	}

}

func GetSPP(c chan definitions.ZincRecordV2) {
	var vals []*definitions.Spp
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, ErcotSPP, nil)
	if err != nil {
		log.Println(err)
		return
	}

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return
	}

	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		log.Println(err)
		return
	}

	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		log.Println(err)
		return
	}

	values := SppParser(doc)

	for _, item := range values {
		df := &definitions.Spp{
			Date:      fmt.Sprintf("%v %v", item[0], item[1]),
			HbBusAvg:  toFloat32(item[2]),
			HbHouston: toFloat32(item[3]),
			HbHubAvg:  toFloat32(item[4]),
			HbNorth:   toFloat32(item[5]),
			HbPan:     toFloat32(item[6]),
			HbSouth:   toFloat32(item[7]),
			HbWest:    toFloat32(item[8]),
			LzAen:     toFloat32(item[9]),
			LzCps:     toFloat32(item[10]),
			LzHouston: toFloat32(item[11]),
			LzLcra:    toFloat32(item[12]),
			LzNorth:   toFloat32(item[13]),
			LzRaybn:   toFloat32(item[14]),
			LzSouth:   toFloat32(item[15]),
			LzWest:    toFloat32(item[16]),
		}
		vals = append(vals, df)
	}
	var envelope []map[string]interface{}
	var tmp map[string]interface{}
	if len(vals) < 1 {
		c <- definitions.ZincRecordV2{
			Index:   "EmptyData",
			Records: envelope,
		}
		return
	}
	out, err := json.Marshal(vals[len(vals)-1])
	if err != nil {
		log.Println(err)
		return
	}
	json.Unmarshal(out, &tmp)
	envelope = append(envelope, tmp)

	c <- definitions.ZincRecordV2{
		Index:   "ErcotSPP",
		Records: envelope,
	}
}

func GetWeather(c chan definitions.ZincRecordV2) {
	cities := []string{"houston", "galveston", "dallas", "austin", "odessa"}
	var wg sync.WaitGroup
	var vals []*definitions.WeatherResponse
	for _, i := range cities {
		var val definitions.WeatherResponse
		wg.Add(1)
		go func(i string) {
			defer wg.Done()
			client := &http.Client{}
			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf(WeatherUri, i), nil)
			if err != nil {
				log.Println(err)
				return
			}
			res, err := client.Do(req)
			if err != nil {
				log.Println(err)
				return
			}
			defer res.Body.Close()

			x, err := io.ReadAll(res.Body)
			if err != nil {
				log.Println(err)
				return
			}
			err = json.Unmarshal(x, &val)
			if err != nil {
				log.Println(err)
				return
			}
			vals = append(vals, &val)
		}(i)
	}
	wg.Wait()
	var envelope []map[string]interface{}
	for _, i := range vals {
		if i.Location == nil {
			continue
		}
		var tmp map[string]interface{}
		out, err := json.Marshal(i)
		if err != nil {
			log.Println(err)
			return
		}
		json.Unmarshal(out, &tmp)
		envelope = append(envelope, tmp)
	}

	c <- definitions.ZincRecordV2{
		Index:   "verySpecialWeather",
		Records: envelope,
	}
}

func CpuMon(c chan definitions.ZincRecordV2) {
	stream := make(chan []*performance.CpuUsage)
	go performance.GetCpuValues(stream, 2)
	msg := <-stream
	var envelope []map[string]interface{}
	for _, i := range msg {
		var tmp map[string]interface{}
		out, err := json.Marshal(i)
		if err != nil {
			log.Println(err)
			return
		}
		json.Unmarshal(out, &tmp)
		envelope = append(envelope, tmp)
	}
	c <- definitions.ZincRecordV2{
		Index:   "cpuMonRxlx",
		Records: envelope,
	}
}

func PowerMonitor(c chan definitions.ZincRecordV2) {
	var container struct {
		Spp     []map[string]interface{} `json:"prices"`
		Rtsc    []map[string]interface{} `json:"system"`
		Weather []map[string]interface{} `json:"weather"`
	}
	newChan := make(chan definitions.ZincRecordV2)
	var vals []map[string]interface{}
	go GetSPP(newChan)
	msg := <-newChan
	container.Spp = append(container.Spp, msg.Records...)
	go GetRealTimeSysCon(newChan)
	msg = <-newChan
	container.Rtsc = append(container.Rtsc, msg.Records...)
	go GetWeather(newChan)
	msg = <-newChan
	container.Weather = append(container.Weather, msg.Records...)

	var tmp map[string]interface{}
	out, err := json.Marshal(container)
	if err != nil {
		log.Println(err)
	}
	json.Unmarshal(out, &tmp)
	vals = append(vals, tmp)
	c <- definitions.ZincRecordV2{
		Index:   "PowerMonitor",
		Records: vals,
	}
}
