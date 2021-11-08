package noaa_weather_api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/testutil"
	"github.com/stretchr/testify/require"
)

const sampleNoContent = `
{
}
`

const sampleStatusResponse = `
{
  "@context": {
    "@version": "1.1",
    "wx": "https://api.weather.gov/ontology#",
    "s": "https://schema.org/",
    "geo": "http://www.opengis.net/ont/geosparql#",
    "unit": "http://codes.wmo.int/common/unit/",
    "@vocab": "https://api.weather.gov/ontology#",
    "geometry": {
      "@id": "s:GeoCoordinates",
      "@type": "geo:wktLiteral"
    },
    "city": "s:addressLocality",
    "state": "s:addressRegion",
    "distance": {
      "@id": "s:Distance",
      "@type": "s:QuantitativeValue"
    },
    "bearing": {
      "@type": "s:QuantitativeValue"
    },
    "value": {
      "@id": "s:value"
    },
    "unitCode": {
      "@id": "s:unitCode",
      "@type": "@id"
    },
    "forecastOffice": {
      "@type": "@id"
    },
    "forecastGridData": {
      "@type": "@id"
    },
    "publicZone": {
      "@type": "@id"
    },
    "county": {
      "@type": "@id"
    }
  },
  "@id": "https://api.weather.gov/stations/KSUA/observations/2021-11-07T18:50:00+00:00",
  "@type": "wx:ObservationStation",
  "geometry": "POINT(-80.22 27.18)",
  "elevation": {
    "unitCode": "wmoUnit:m",
    "value": 6
  },
  "station": "https://api.weather.gov/stations/KSUA",
  "timestamp": "2021-11-07T18:50:00+00:00",
  "rawMessage": "KSUA 071850Z 34012G21KT 10SM FEW075 21/11 A2998",
  "textDescription": "Mostly Clear",
  "icon": "https://api.weather.gov/icons/land/day/few?size=medium",
  "presentWeather": [],
  "temperature": {
    "unitCode": "wmoUnit:degC",
    "value": 21,
    "qualityControl": "V"
  },
  "dewpoint": {
    "unitCode": "wmoUnit:degC",
    "value": 11,
    "qualityControl": "V"
  },
  "windDirection": {
    "unitCode": "wmoUnit:degree_(angle)",
    "value": 340,
    "qualityControl": "V"
  },
  "windSpeed": {
    "unitCode": "wmoUnit:km_h-1",
    "value": 22.32,
    "qualityControl": "V"
  },
  "windGust": {
    "unitCode": "wmoUnit:km_h-1",
    "value": 38.88,
    "qualityControl": "S"
  },
  "barometricPressure": {
    "unitCode": "wmoUnit:Pa",
    "value": 101520,
    "qualityControl": "V"
  },
  "seaLevelPressure": {
    "unitCode": "wmoUnit:Pa",
    "value": null,
    "qualityControl": "Z"
  },
  "visibility": {
    "unitCode": "wmoUnit:m",
    "value": 16090,
    "qualityControl": "C"
  },
  "maxTemperatureLast24Hours": {
    "unitCode": "wmoUnit:degC",
    "value": null
  },
  "minTemperatureLast24Hours": {
    "unitCode": "wmoUnit:degC",
    "value": null
  },
  "precipitationLastHour": {
    "unitCode": "wmoUnit:m",
    "value": null,
    "qualityControl": "Z"
  },
  "precipitationLast3Hours": {
    "unitCode": "wmoUnit:m",
    "value": null,
    "qualityControl": "Z"
  },
  "precipitationLast6Hours": {
    "unitCode": "wmoUnit:m",
    "value": null,
    "qualityControl": "Z"
  },
  "relativeHumidity": {
    "unitCode": "wmoUnit:percent",
    "value": 52.802638324228,
    "qualityControl": "V"
  },
  "windChill": {
    "unitCode": "wmoUnit:degC",
    "value": null,
    "qualityControl": "V"
  },
  "heatIndex": {
    "unitCode": "wmoUnit:degC",
    "value": null,
    "qualityControl": "V"
  },
  "cloudLayers": [
    {
      "base": {
        "unitCode": "wmoUnit:m",
        "value": 2290
      },
      "amount": "FEW"
    }
  ]
}
`

func TestWeatherGeneratesMetrics(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var rsp string
		if r.URL.Path == "/stations/KSUA/observations/latest" {
			rsp = sampleStatusResponse
			w.Header()["Content-Type"] = []string{"application/ld+json"}
		} else {
			require.Fail(t, "Cannot handle request")
		}

		_, err := fmt.Fprintln(w, rsp)
		require.NoError(t, err)
	}))
	defer ts.Close()

	n := &NOAAWeatherAPI{
		BaseURL:   ts.URL,
		StationID: []string{"KSUA"},
		Units:     "metric",
	}
	require.NoError(t, n.Init())

	var acc testutil.Accumulator

	require.NoError(t, n.Gather(&acc))

	expected := []telegraf.Metric{
		testutil.MustMetric(
			"weather",
			map[string]string{
				"station": "KSUA",
			},
			map[string]interface{}{
				"temperature":    float64(21),
				"humidity":       float64(52.802638324228),
				"pressure":       float64(101520),
				"visibility":     float64(16090),
				"dewpoint":       float64(11),
				"wind_speed":     float64(22.32),
				"wind_degrees":   float64(340),
			},
			time.Unix(1636311000, 0),
		),
	}
	testutil.RequireMetricsEqual(t, expected, acc.GetTelegrafMetrics())
}



func TestWeatherGeneratesImperial(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var rsp string
		if r.URL.Path == "/stations/KSUA/observations/latest" {
			rsp = sampleStatusResponse
			w.Header()["Content-Type"] = []string{"application/ld+json"}
		} else {
			require.Fail(t, "Cannot handle request")
		}

		_, err := fmt.Fprintln(w, rsp)
		require.NoError(t, err)
	}))
	defer ts.Close()

	n := &NOAAWeatherAPI{
		BaseURL:   ts.URL,
		StationID: []string{"KSUA"},
		Units:     "imperial",
	}
	require.NoError(t, n.Init())

	var acc testutil.Accumulator

	require.NoError(t, n.Gather(&acc))

	expected := []telegraf.Metric{
		testutil.MustMetric(
			"weather",
			map[string]string{
				"station": "KSUA",
			},
			map[string]interface{}{
				"temperature":    float64(69.8),
				"humidity":       float64(52.802638324228),
				"pressure":       float64(101520),
				"visibility":     float64(10),
				"dewpoint":       float64(11),
				"wind_speed":     float64(13.871970167806092),
				"wind_degrees":   float64(340),
			},
			time.Unix(1636311000, 0),
		),
	}
	testutil.RequireMetricsEqual(t, expected, acc.GetTelegrafMetrics())
}



func TestWeatherGeneratesImperialMultiple(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var rsp string
		if r.URL.Path == "/stations/KSUA/observations/latest" {
			rsp = sampleStatusResponse
			w.Header()["Content-Type"] = []string{"application/ld+json"}
		} else {
			require.Fail(t, "Cannot handle request")
		}

		_, err := fmt.Fprintln(w, rsp)
		require.NoError(t, err)
	}))
	defer ts.Close()

	n := &NOAAWeatherAPI{
		BaseURL:   ts.URL,
		StationID: []string{"KSUA", "KSUA"},
		Units:     "imperial",
	}
	require.NoError(t, n.Init())

	var acc testutil.Accumulator

	require.NoError(t, n.Gather(&acc))

	expected := []telegraf.Metric{
		testutil.MustMetric(
			"weather",
			map[string]string{
				"station": "KSUA",
			},
			map[string]interface{}{
				"temperature":    float64(69.8),
				"humidity":       float64(52.802638324228),
				"pressure":       float64(101520),
				"visibility":     float64(10),
				"dewpoint":       float64(11),
				"wind_speed":     float64(13.871970167806092),
				"wind_degrees":   float64(340),
			},
			time.Unix(1636311000, 0),
		),
		testutil.MustMetric(
			"weather",
			map[string]string{
				"station": "KSUA",
			},
			map[string]interface{}{
				"temperature":    float64(69.8),
				"humidity":       float64(52.802638324228),
				"pressure":       float64(101520),
				"visibility":     float64(10),
				"dewpoint":       float64(11),
				"wind_speed":     float64(13.871970167806092),
				"wind_degrees":   float64(340),
			},
			time.Unix(1636311000, 0),
		),		
	}
	testutil.RequireMetricsEqual(t, expected, acc.GetTelegrafMetrics())
}


func TestFormatURL(t *testing.T) {
	n := &NOAAWeatherAPI{
		Units:   "metric",
		BaseURL: "http://foo.com",
	}
	require.NoError(t, n.Init())

	require.Equal(t,
		"http://foo.com/stations/KSUA/observations/latest?require_qc=false",
		n.formatURL("/stations/%s/observations/latest", "KSUA"))
}

func TestDefaultUnits(t *testing.T) {
	n := &NOAAWeatherAPI{}
	require.NoError(t, n.Init())

	require.Equal(t, "metric", n.Units)
}


