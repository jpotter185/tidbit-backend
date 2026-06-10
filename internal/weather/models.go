package weather

type WeatherRequest struct {
    Latitude  float64
    Longitude float64
}

type openMeteoResponse struct {
    UtcOffsetSeconds int                `json:"utc_offset_seconds"`
    Current          openMeteoCurrent   `json:"current"`
    Daily            openMeteoDaily     `json:"daily"`
    Hourly           openMeteoHourly    `json:"hourly"`
}

type openMeteoCurrent struct {
    Time                string  `json:"time"`
    Temperature         float64 `json:"temperature_2m"`
    ApparentTemperature float64 `json:"apparent_temperature"`
    IsDay               int     `json:"is_day"`
    WeatherCode         int     `json:"weather_code"`
    Humidity            float64 `json:"relative_humidity_2m"`
}

type openMeteoDaily struct {
    MaxTemp []float64 `json:"temperature_2m_max"`
    MinTemp []float64 `json:"temperature_2m_min"`
}

type openMeteoHourly struct {
    UVIndex              []float64 `json:"uv_index"`
    PrecipProbability    []float64 `json:"precipitation_probability"`
}

type WeatherResponse struct {
    CurrentTemperature  int `json:"current_temperature"`
    FeelsLike           int `json:"feels_like_temperature"`
    IsDay               int `json:"is_day"`
    WeatherCode         int `json:"weather_code"`
    Humidity            int `json:"humidity"`
    MinTemp             int `json:"min_temp"`
    MaxTemp             int `json:"max_temp"`
    PrecipProbability   int `json:"precip_probability"`
    UVIndex             int `json:"uv_index"`
    TZOffset            int `json:"tz_offset"`
}