package weather

import (
    "encoding/json"
    "fmt"
    "math"
    "net/http"
    "strconv"
    "time"
)

const (
    requestTimeout = 10 * time.Second
    maxAttempts    = 3
    retryDelay     = 500 * time.Millisecond
)

type WeatherClient struct {
    BaseURL    string
    HTTPClient *http.Client
}

func NewWeatherClient() *WeatherClient {
    return &WeatherClient{
        BaseURL:    "https://api.open-meteo.com/v1/forecast",
        HTTPClient: &http.Client{Timeout: requestTimeout},
    }
}

func (w *WeatherClient) GetWeather(req WeatherRequest) (*WeatherResponse, error) {
    url := fmt.Sprintf(
        "%s?latitude=%s&longitude=%s&current=temperature_2m,apparent_temperature,is_day,weather_code,relative_humidity_2m&daily=temperature_2m_max,temperature_2m_min&hourly=uv_index,precipitation_probability&timezone=auto&forecast_days=1&wind_speed_unit=mph&temperature_unit=fahrenheit&precipitation_unit=inch",
        w.BaseURL,
        strconv.FormatFloat(req.Latitude, 'f', 4, 64),
        strconv.FormatFloat(req.Longitude, 'f', 4, 64),
    )

    var lastErr error
    for attempt := 1; attempt <= maxAttempts; attempt++ {
        raw, err := w.fetch(url)
        if err == nil {
            var parsed *WeatherResponse
            parsed, err = parseResponse(raw)
            if err == nil {
                return parsed, nil
            }
        }
        lastErr = err
        if attempt < maxAttempts {
            time.Sleep(retryDelay * time.Duration(attempt))
        }
    }

    return nil, lastErr
}

func (w *WeatherClient) fetch(url string) (openMeteoResponse, error) {
    var raw openMeteoResponse

    resp, err := w.HTTPClient.Get(url)
    if err != nil {
        return raw, fmt.Errorf("failed to call open-meteo: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return raw, fmt.Errorf("unexpected status from open-meteo: %d", resp.StatusCode)
    }

    if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
        return raw, fmt.Errorf("failed to decode response: %w", err)
    }

    return raw, nil
}

func parseResponse(raw openMeteoResponse) (*WeatherResponse, error) {
    // Current.Time is ISO 8601 local time ("2026-07-19T21:00"); the hour
    // indexes into the hourly arrays because timezone=auto keeps them local.
    if len(raw.Current.Time) < 13 {
        return nil, fmt.Errorf("unexpected current time from open-meteo: %q", raw.Current.Time)
    }
    currentHour, err := strconv.Atoi(raw.Current.Time[11:13])
    if err != nil || currentHour < 0 || currentHour > 23 {
        return nil, fmt.Errorf("unexpected current time from open-meteo: %q", raw.Current.Time)
    }
    if len(raw.Daily.MinTemp) == 0 || len(raw.Daily.MaxTemp) == 0 {
        return nil, fmt.Errorf("missing daily data from open-meteo")
    }
    if currentHour >= len(raw.Hourly.PrecipProbability) || currentHour >= len(raw.Hourly.UVIndex) {
        return nil, fmt.Errorf("missing hourly data from open-meteo for hour %d", currentHour)
    }

    return &WeatherResponse{
        CurrentTemperature: round(raw.Current.Temperature),
        FeelsLike:          round(raw.Current.ApparentTemperature),
        IsDay:              raw.Current.IsDay,
        WeatherCode:        raw.Current.WeatherCode,
        Humidity:           round(raw.Current.Humidity),
        MinTemp:            round(raw.Daily.MinTemp[0]),
        MaxTemp:            round(raw.Daily.MaxTemp[0]),
        PrecipProbability:  round(raw.Hourly.PrecipProbability[currentHour]),
        UVIndex:            round(raw.Hourly.UVIndex[currentHour]),
        TZOffset:           raw.UtcOffsetSeconds / 3600,
    }, nil
}

// round converts to the nearest integer rather than truncating toward
// zero, so 72.9 displays as 73 and -5.9 as -6.
func round(f float64) int {
    return int(math.Round(f))
}