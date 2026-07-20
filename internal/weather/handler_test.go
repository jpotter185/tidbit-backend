package weather

import "testing"

func validResponse() openMeteoResponse {
	hourly := make([]float64, 24)
	return openMeteoResponse{
		UtcOffsetSeconds: -14400,
		Current: openMeteoCurrent{
			Time:        "2026-07-19T21:00",
			Temperature: 71.3,
			Humidity:    51,
		},
		Daily: openMeteoDaily{
			MaxTemp: []float64{81.1},
			MinTemp: []float64{66.2},
		},
		Hourly: openMeteoHourly{
			UVIndex:           hourly,
			PrecipProbability: hourly,
		},
	}
}

func TestParseResponseValid(t *testing.T) {
	resp, err := parseResponse(validResponse())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.CurrentTemperature != 71 || resp.MinTemp != 66 || resp.MaxTemp != 81 || resp.TZOffset != -4 {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestParseResponseMalformed(t *testing.T) {
	cases := map[string]func(*openMeteoResponse){
		"empty body":         func(r *openMeteoResponse) { *r = openMeteoResponse{} },
		"short time":         func(r *openMeteoResponse) { r.Current.Time = "2026-07-19" },
		"non-numeric hour":   func(r *openMeteoResponse) { r.Current.Time = "2026-07-19Txx:00" },
		"empty daily":        func(r *openMeteoResponse) { r.Daily = openMeteoDaily{} },
		"short hourly":       func(r *openMeteoResponse) { r.Hourly.UVIndex = r.Hourly.UVIndex[:2] },
		"empty hourly":       func(r *openMeteoResponse) { r.Hourly = openMeteoHourly{} },
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			raw := validResponse()
			mutate(&raw)
			if _, err := parseResponse(raw); err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}
