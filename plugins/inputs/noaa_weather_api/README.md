# NOAA Weather API Input Plugin

Collect current weather and forecast data from NOAA Weather API.

Station idenifiers can be found in the [noaa weather api][].

### Configuration

```toml
[[inputs.noaa_weather_api]]
  ## NOAA Weather API

  ## Stations to collect weather data from.
  station_id = ["KSUA"]

  ## base URL
  # base_url = "https://api.weather.gov"

  ## Timeout for HTTP response.
  # response_timeout = "5s"

  ## Preferred unit system for temperature and wind speed. Can be one of
  ## "metric" or "imperial".
  # units = "metric"

  ## Query interval;
  ## minutes.
  interval = "10m"
  
  ## UserAgent
  user_agent = "You Server name you@email.com"
```

### Metrics

- weather
  - tags:
    - station
  - fields:
    - humidity (float, percent)
    - pressure (float, atmospheric pressure hPa)
    - temperature (float, degrees)
    - visibility (int, meters)
    - wind_degrees (float, wind direction in degrees)
    - wind_speed (float, wind speed in km/hr or miles/hr)

### Example Output

```

```

[noaa weather api]:https://www.weather.gov/documentation/services-web-api#/

