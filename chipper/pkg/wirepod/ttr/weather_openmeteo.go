package wirepod_ttr

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kercre123/wire-pod/chipper/pkg/logger"
)

var openMeteoHTTP = &http.Client{Timeout: 5 * time.Second}

type openMeteoCity struct {
	Name        string  `json:"name"`
	CountryCode string  `json:"country_code"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

type openMeteoReading struct {
	Time        int64    `json:"time"`
	Temperature *float64 `json:"temperature_2m"`
	Code        *int     `json:"weather_code"`
	IsDay       int      `json:"is_day"`
}

func getOpenMeteoJSON(endpoint string, params url.Values, dst any) error {
	resp, err := openMeteoHTTP.Get(endpoint + "?" + params.Encode())
	if err != nil {
		return fmt.Errorf("weather connection failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("weather service returned HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(dst)
}

func findOpenMeteoCity(location string) (openMeteoCity, error) {
	name, region, _ := strings.Cut(location, ",")
	name, region = strings.TrimSpace(name), strings.TrimSpace(region)
	if name == "北京" || name == "北京市" || (strings.EqualFold(name, "Beijing") && (region == "" || strings.EqualFold(region, "CN"))) {
		return openMeteoCity{Name: "Beijing", CountryCode: "CN", Latitude: 39.9042, Longitude: 116.4074}, nil
	}
	if name == "" {
		return openMeteoCity{}, fmt.Errorf("no weather city configured")
	}
	var data struct {
		Results []openMeteoCity `json:"results"`
	}
	params := url.Values{"name": {name}, "count": {"10"}, "language": {"en"}, "format": {"json"}}
	if len(region) == 2 {
		params.Set("countryCode", strings.ToUpper(region))
	}
	if err := getOpenMeteoJSON("https://geocoding-api.open-meteo.com/v1/search", params, &data); err != nil {
		return openMeteoCity{}, err
	}
	if len(data.Results) == 0 || data.Results[0].Name == "" {
		return openMeteoCity{}, fmt.Errorf("no matching weather city")
	}
	return data.Results[0], nil
}

func openMeteoCondition(code int, isDay bool) string {
	switch code {
	case 0, 1:
		if isDay {
			return "Sunny"
		}
		return "Stars"
	case 2:
		return "PartlyCloudy"
	case 3:
		return "Cloudy"
	case 45, 48:
		return "Fog"
	case 51, 53, 55, 56, 57, 61, 63, 65, 66, 67, 80, 81, 82:
		return "Rain"
	case 71, 73, 75, 77, 85, 86:
		return "Snow"
	case 95, 96, 97, 99:
		return "Thunderstorms"
	default:
		return ""
	}
}

func getOpenMeteoWeather(location, unit string, hoursFromNow int) (string, string, string, string, string, string) {
	errorResult := func(err error) (string, string, string, string, string, string) {
		logger.Println("Open-Meteo weather failed: ", err)
		return "undefined", "false", "test", location, "", unit
	}
	city, err := findOpenMeteoCity(location)
	if err != nil {
		return errorResult(err)
	}
	if unit != "F" {
		unit = "C"
	}
	tempUnit := "celsius"
	if unit == "F" {
		tempUnit = "fahrenheit"
	}
	forecast := hoursFromNow > 0
	params := url.Values{
		"latitude": {strconv.FormatFloat(city.Latitude, 'f', 4, 64)}, "longitude": {strconv.FormatFloat(city.Longitude, 'f', 4, 64)},
		"timezone": {"auto"}, "timeformat": {"unixtime"}, "temperature_unit": {tempUnit},
		"current": {"temperature_2m,weather_code,is_day"},
	}
	if forecast {
		params.Set("hourly", "temperature_2m,weather_code,is_day")
		params.Set("forecast_days", "4")
	}
	var data struct {
		UTCOffset int              `json:"utc_offset_seconds"`
		Current   openMeteoReading `json:"current"`
		Hourly    struct {
			Times        []int64    `json:"time"`
			Temperatures []*float64 `json:"temperature_2m"`
			Codes        []*int     `json:"weather_code"`
			IsDay        []int      `json:"is_day"`
		} `json:"hourly"`
	}
	if err := getOpenMeteoJSON("https://api.open-meteo.com/v1/forecast", params, &data); err != nil {
		return errorResult(err)
	}
	reading := data.Current
	if forecast {
		target := time.Now().Add(time.Duration(hoursFromNow) * time.Hour).Truncate(time.Hour).Unix()
		reading = openMeteoReading{}
		for i, tick := range data.Hourly.Times {
			if tick >= target && i < len(data.Hourly.Temperatures) && i < len(data.Hourly.Codes) && i < len(data.Hourly.IsDay) {
				reading = openMeteoReading{Time: tick, Temperature: data.Hourly.Temperatures[i], Code: data.Hourly.Codes[i], IsDay: data.Hourly.IsDay[i]}
				break
			}
		}
	}
	if reading.Time == 0 || reading.Temperature == nil || reading.Code == nil {
		return errorResult(fmt.Errorf("weather response was missing data"))
	}
	condition := openMeteoCondition(*reading.Code, reading.IsDay == 1)
	if condition == "" {
		return errorResult(fmt.Errorf("unrecognized weather code %d", *reading.Code))
	}
	if condition == "PartlyCloudy" || condition == "Fog" {
		condition = "Cloudy"
	}
	date := time.Unix(reading.Time, 0).In(time.FixedZone("", data.UTCOffset)).Format("2006-01-02 15:04")
	logger.LogUI("Open-Meteo weather retrieved for " + city.Name)
	return condition, strconv.FormatBool(forecast), date, city.Name, strconv.Itoa(int(math.Round(*reading.Temperature))), unit
}

func openMeteoChineseLocation(speech string) string {
	before, _, found := strings.Cut(strings.Join(strings.Fields(speech), ""), "天气")
	if !found {
		return ""
	}
	before = strings.NewReplacer(
		"帮我查一下", "", "帮我看看", "", "请问", "", "查一下", "", "告诉我", "",
		"今天", "", "明天", "", "后天", "", "现在", "", "今晚", "", "上午", "", "下午", "",
	).Replace(before)
	return strings.Trim(before, " ，。？！的")
}
