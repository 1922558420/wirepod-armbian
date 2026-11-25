package vars

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/net/http/httpguts"
)

type CustomWeatherConfig struct {
	Endpoint        string `json:"endpoint"`
	Location        string `json:"location"`
	Unit            string `json:"unit"`
	Header          string `json:"header"`
	QueryKey        string `json:"query_key"`
	Secret          string `json:"secret,omitempty"`
	TemperaturePath string `json:"temperature_path"`
	ConditionPath   string `json:"condition_path"`
	ConditionMap    string `json:"condition_map"`
}

func (c *CustomWeatherConfig) WithoutSecret() *CustomWeatherConfig {
	if c == nil {
		return nil
	}
	public := *c
	public.Secret = ""
	return &public
}

func (c CustomWeatherConfig) RequestURL(location, unit, hours string) (string, error) {
	origin, err := url.Parse(c.Endpoint)
	if err != nil || origin.Hostname() == "" || origin.User != nil || origin.Fragment != "" ||
		strings.ContainsAny(origin.Host, "{}") || (origin.Scheme != "https" && origin.Scheme != "http") {
		return "", fmt.Errorf("enter an HTTP or HTTPS weather URL without embedded credentials or a fragment")
	}
	tokens := strings.NewReplacer("{location}", url.QueryEscape(location), "{unit}", unit, "{hours}", hours)
	endpoint := tokens.Replace(c.Endpoint)
	if c.QueryKey == "" || c.Secret == "" {
		return endpoint, nil
	}
	target, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("invalid custom weather request")
	}
	query := target.Query()
	query.Set(c.QueryKey, c.Secret)
	target.RawQuery = query.Encode()
	return target.String(), nil
}

func (c CustomWeatherConfig) SameAuthenticationTarget(other CustomWeatherConfig) bool {
	first, err := url.Parse(c.Endpoint)
	second, otherErr := url.Parse(other.Endpoint)
	return err == nil && otherErr == nil && first.Scheme == second.Scheme && strings.EqualFold(first.Host, second.Host) &&
		strings.EqualFold(c.Header, other.Header) && c.QueryKey == other.QueryKey
}

func (c CustomWeatherConfig) Mapping() (map[string]string, error) {
	mapping := make(map[string]string)
	if strings.TrimSpace(c.ConditionMap) != "" {
		if err := json.Unmarshal([]byte(c.ConditionMap), &mapping); err != nil || mapping == nil {
			return nil, fmt.Errorf("condition mapping must be a JSON object with string values")
		}
	}
	return mapping, nil
}

func (c CustomWeatherConfig) Validate() error {
	if _, err := c.RequestURL("test", "C", "0"); err != nil {
		return err
	}
	if c.Unit != "" && c.Unit != "C" && c.Unit != "F" {
		return fmt.Errorf("choose Celsius, Fahrenheit, or the robot's unit")
	}
	for _, pointer := range []string{c.TemperaturePath, c.ConditionPath} {
		if pointer != "" && !strings.HasPrefix(pointer, "/") {
			return fmt.Errorf("JSON field paths must start with /")
		}
	}
	if strings.EqualFold(c.Header, "Host") || (c.Header != "" && !httpguts.ValidHeaderFieldName(c.Header)) ||
		!httpguts.ValidHeaderFieldValue(c.Secret) {
		return fmt.Errorf("invalid weather authentication header")
	}
	if c.Header != "" && c.QueryKey != "" {
		return fmt.Errorf("choose one authentication field")
	}
	_, err := c.Mapping()
	return err
}
