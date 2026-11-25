package wirepod_ttr

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kercre123/wire-pod/chipper/pkg/vars"
)

func TestCustomWeather(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("city") != "北京 & 周边" || q.Get("hours") != "24" || q.Get("unit") != "C" ||
			r.Header.Get("X-Weather-Token") != "test-value" {
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		fmt.Fprint(w, `{"forecast":[{"temp":"-2.4","code":61}]}`)
	}))
	defer server.Close()
	config := vars.CustomWeatherConfig{
		Endpoint: server.URL + "?city={location}&hours={hours}&unit={unit}",
		Location: "北京 & 周边", Unit: "C", Header: "X-Weather-Token", Secret: "test-value",
		TemperaturePath: "/forecast/0/temp", ConditionPath: "/forecast/0/code",
		ConditionMap: `{"61":"Rain"}`,
	}
	if err := config.Validate(); err != nil {
		t.Fatal(err)
	}
	condition, forecast, date, city, temperature, units := getCustomWeather(config, "ignored", "F", 24)
	if condition != "Rain" || forecast != "true" || date == "test" || city != config.Location || temperature != "-2" || units != "C" {
		t.Fatalf("unexpected firmware weather data: %q %q %q %q %q %q", condition, forecast, date, city, temperature, units)
	}
}

func TestCustomWeatherRejectsRedirects(t *testing.T) {
	main := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/unexpected" {
			t.Error("followed a redirect")
			return
		}
		http.Redirect(w, r, "/unexpected", http.StatusFound)
	}))
	defer main.Close()
	_, _, err := fetchCustomWeather(vars.CustomWeatherConfig{Endpoint: main.URL}, "Beijing", "C", 0)
	if err == nil {
		t.Fatal("accepted redirect as weather response")
	}
}
