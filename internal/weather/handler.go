package weather

import (
    "encoding/json"
    "fmt"
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
            return parseResponse(raw), nil
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

func parseResponse(raw openMeteoResponse) *WeatherResponse {
    currentHour, _ := strconv.Atoi(raw.Current.Time[11:13])

    return &WeatherResponse{
        CurrentTemperature: int(raw.Current.Temperature),
        FeelsLike:          int(raw.Current.ApparentTemperature),
        IsDay:              raw.Current.IsDay,
        WeatherCode:        raw.Current.WeatherCode,
        Humidity:           int(raw.Current.Humidity),
        MinTemp:            int(raw.Daily.MinTemp[0]),
        MaxTemp:            int(raw.Daily.MaxTemp[0]),
        PrecipProbability:  int(raw.Hourly.PrecipProbability[currentHour]),
        UVIndex:            int(raw.Hourly.UVIndex[currentHour]),
        TZOffset:           raw.UtcOffsetSeconds / 3600,
    }
}