package noaa_weather_api

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/config"
	"github.com/influxdata/telegraf/plugins/inputs"
)

// https://www.weather.gov/documentation/services-web-api#/default/station_observation_latest

const (
	nwaRequestSeveralStationID int = 20
	defaultStationId               = "KSUA"
	defaultBaseURL                 = "https://api.weather.gov/"
	defaultResponseTimeout         = time.Second * 5
	defaultUnits                   = "imperial"
)

type NOAAWeatherAPI struct {
	StationID       []string        `toml:"station_id"`
	BaseURL         string          `toml:"base_url"`
	ResponseTimeout config.Duration `toml:"response_timeout"`
	Units           string          `toml:"units"`
	UserAgent       string          `toml:"user_agent"`
	client          *http.Client
	baseParsedURL   *url.URL
}

var sampleConfig = `
  ## NOAA Weather API

  ## Stations to collect weather data from.
  station_id = ["KSUA"]

  ## base URL
  # base_url = "https://api.weather.gov"

  ## Timeout for HTTP response.
  # response_timeout = "5s"

  ## Preferred unit system for temperature and wind speed. Can be one of
  ## "metric" or "imperial".
  # units = "imperial"

  ## Query interval;
  ## minutes.
  interval = "10m"
  
  ## UserAgent
  user_agent = "Your Server name <you@email.com>"
`

func (n *NOAAWeatherAPI) SampleConfig() string {
	return sampleConfig
}

func (n *NOAAWeatherAPI) Description() string {
	return "Read current weather from NOAA Weather API"
}

func (n *NOAAWeatherAPI) Gather(acc telegraf.Accumulator) error {
	var wg sync.WaitGroup

	for _, station := range n.StationID {
		addr := n.formatURL("/stations/%s/observations/latest", station)
		wg.Add(1)
		go func() {
			defer wg.Done()
			status, err := n.gatherURL(addr)
			if err != nil {
				acc.AddError(err)
				return
			}

			n.GatherWeather(acc, status)
		}()
	}

	wg.Wait()
	return nil
}

func (n *NOAAWeatherAPI) createHTTPClient() *http.Client {
	if n.ResponseTimeout < config.Duration(time.Second) {
		n.ResponseTimeout = config.Duration(defaultResponseTimeout)
	}

	client := &http.Client{
		Transport: &http.Transport{},
		Timeout:   time.Duration(n.ResponseTimeout),
	}

	return client
}

func (n *NOAAWeatherAPI) gatherURL(addr string) (*Status, error) {
	req, err := http.NewRequest("GET", addr, nil)
	req.Header.Add("Accept", "application/ld+json")
	req.Header.Add("User-Agent", n.UserAgent)
	resp, err := n.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making HTTP request to %s: %s", addr, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned HTTP status %s", addr, resp.Status)
	}

	mediaType, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}

	if mediaType != "application/ld+json" {
		return nil, fmt.Errorf("%s returned unexpected content type %s", addr, mediaType)
	}

	return gatherWeatherURL(resp.Body)
}

type ApiValue struct {
	UnitCode       string  `json:"unitCode"`
	Value          float64 `json:"value"`
	QualityControl string  `json:"qualityControl"`
}

type Status struct {
	Temperature        ApiValue `json:"temperature"`
	Humidity           ApiValue `json:"relativeHumidity"`
	BarometricPressure ApiValue `json:"barometricPressure"`
	Visibility         ApiValue `json:"visibility"`
	WindSpeed          ApiValue `json:"windSpeed"`
	WindDirection      ApiValue `json:"windDirection"`
	Dewpoint           ApiValue `json:"dewpoint"`
	Timestamp          string   `json:"timestamp"`
}

func gatherWeatherURL(r io.Reader) (*Status, error) {
	dec := json.NewDecoder(r)
	status := &Status{}
	if err := dec.Decode(status); err != nil {
		return nil, fmt.Errorf("error while decoding JSON response: %s", err)
	}
	return status, nil
}

func (n *NOAAWeatherAPI) UnitConversion(value ApiValue) float64 {

	switch value.UnitCode {
	case "wmoUnit:degC":
		if n.Units == "imperial" {
			return value.Value*9.0/5.0 + 32
		} else {
			return value.Value
		}
	case "wmoUnit:km_h-1":
		if n.Units == "imperial" {
			return value.Value / 1.609
		} else {
			return value.Value
		}
	case "wmoUnit:m":
		if n.Units == "imperial" {
			return value.Value / 1609.0
		} else {
			return value.Value
		}
	default:
		return value.Value
	}
}

func (n *NOAAWeatherAPI) GatherWeather(acc telegraf.Accumulator, status *Status) {
	fields := map[string]interface{}{
		"pressure":     status.BarometricPressure.Value,
		"dewpoint":     status.Dewpoint.Value,
		"temperature":  n.UnitConversion(status.Temperature),
		"humidity":     status.Humidity.Value,
		"visibility":   n.UnitConversion(status.Visibility),
		"wind_degrees": status.WindDirection.Value,
		"wind_speed":   n.UnitConversion(status.WindSpeed),
	}
	tags := map[string]string{
		"station": "KSUA",
	}

	layout := "2006-01-02T15:04:05Z07:00"
	tm, err := time.Parse(layout, status.Timestamp)
	if err != nil {
		fmt.Errorf("%s", err)
	} else {
		acc.AddFields("noaa_weather", fields, tags, tm)
	}
}

func init() {
	inputs.Add("noaa_weather_api", func() telegraf.Input {
		tmout := config.Duration(defaultResponseTimeout)
		return &NOAAWeatherAPI{
			ResponseTimeout: tmout,
			BaseURL:         defaultBaseURL,
		}
	})
}

func (n *NOAAWeatherAPI) Init() error {
	var err error
	n.baseParsedURL, err = url.Parse(n.BaseURL)
	if err != nil {
		return err
	}

	n.client = n.createHTTPClient()

	switch n.Units {
	case "imperial", "metric":
	case "":
		n.Units = defaultUnits
	default:
		return fmt.Errorf("unknown units: %s", n.Units)
	}

	return nil
}

func (n *NOAAWeatherAPI) formatURL(path string, station_id string) string {

	v := url.Values{
		"require_qc": []string{"false"},
	}

	relative := &url.URL{
		Path:     fmt.Sprintf(path, url.PathEscape(station_id)),
		RawQuery: v.Encode(),
	}

	return n.baseParsedURL.ResolveReference(relative).String()
}
