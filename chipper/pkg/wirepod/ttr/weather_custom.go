package wirepod_ttr

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kercre123/wire-pod/chipper/pkg/logger"
	"github.com/kercre123/wire-pod/chipper/pkg/vars"
)

var customWeatherHTTP = &http.Client{
	Timeout: 6 * time.Second,
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

func customWeatherField(body any, pointer, fallback string) any {
	if pointer == "" {
		pointer = fallback
	}
	for _, token := range strings.Split(strings.TrimPrefix(pointer, "/"), "/") {
		token = strings.NewReplacer("~1", "/", "~0", "~").Replace(token)
		switch value := body.(type) {
		case map[string]any:
			body = value[token]
		case []any:
			i, err := strconv.Atoi(token)
			if err != nil || i < 0 || i >= len(value) {
				return nil
			}
			body = value[i]
		default:
			return nil
		}
	}
	return body
}

func customWeatherCondition(text string) string {
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "sunny":
		return "Sunny"
	case "clear":
		return "Clear"
	case "stars":
		return "Stars"
	case "cloudy", "partlycloudy", "fog":
		return "Cloudy"
	case "rain":
		return "Rain"
	case "snow":
		return "Snow"
	case "thunderstorms":
		return "Thunderstorms"
	case "windy":
		return "Windy"
	case "cold":
		return "Cold"
	}
	return ""
}

func fetchCustomWeather(config vars.CustomWeatherConfig, location, unit string, hours int) (string, float64, error) {
	endpoint, err := config.RequestURL(location, unit, strconv.Itoa(hours))
	if err != nil {
		return "", 0, err
	}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", 0, fmt.Errorf("invalid custom weather request")
	}
	req.Header.Set("Accept", "application/json")
	if config.Header != "" && config.Secret != "" {
		req.Header.Set(config.Header, config.Secret)
	}
	resp, err := customWeatherHTTP.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("custom weather connection failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("custom weather returned HTTP %d", resp.StatusCode)
	}
	const maxResponseSize = 1 << 20
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
	if err != nil || len(raw) > maxResponseSize {
		return "", 0, fmt.Errorf("could not read custom weather response")
	}
	var body any
	if json.Unmarshal(raw, &body) != nil {
		return "", 0, fmt.Errorf("custom weather response must be JSON")
	}
	value := customWeatherField(body, config.TemperaturePath, "/temperature")
	temp, err := strconv.ParseFloat(fmt.Sprint(value), 64)
	if err != nil || math.IsNaN(temp) || temp < -150 || temp > 200 {
		return "", 0, fmt.Errorf("custom weather response has invalid temperature")
	}
	code := fmt.Sprint(customWeatherField(body, config.ConditionPath, "/condition"))
	mapping, err := config.Mapping()
	if err != nil {
		return "", 0, err
	}
	if mapped, ok := mapping[code]; ok {
		code = mapped
	}
	condition := customWeatherCondition(code)
	if condition == "" {
		return "", 0, fmt.Errorf("custom weather condition needs a Vector condition mapping")
	}
	return condition, temp, nil
}

func getCustomWeather(config vars.CustomWeatherConfig, location, unit string, hours int) (string, string, string, string, string, string) {
	if config.Location != "" {
		location = config.Location
	}
	if config.Unit != "" {
		unit = config.Unit
	}
	if unit != "F" {
		unit = "C"
	}
	condition, temperature, err := fetchCustomWeather(config, location, unit, hours)
	if err != nil {
		logger.LogUI("Custom weather failed: ", err)
		return "undefined", "false", "test", location, "", unit
	}
	date := time.Now().Add(time.Duration(hours) * time.Hour).Format("2006-01-02 15:04")
	return condition, strconv.FormatBool(hours > 0), date, location, strconv.Itoa(int(math.Round(temperature))), unit
}
