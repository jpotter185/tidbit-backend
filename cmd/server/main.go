package main

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"tidbit-backend/internal/sensor"
	"tidbit-backend/internal/weather"

	"github.com/gin-gonic/gin"
)

// sensorRadiusKm is how close (great-circle) a sensor must be to the
// requested coordinates for its readings to be matched to them.
const sensorRadiusKm = 10.0

// maxSensorAge is how recent a reading must be to override the forecast;
// beyond this the sensor is presumed offline and Open-Meteo data is used.
const maxSensorAge = 1 * time.Hour

// maxLoggedBody caps how much of a request/response body goes into a log
// line (the history endpoint can return large payloads).
const maxLoggedBody = 2048

// maxRequestBody caps how much of a request body the server will buffer.
// Sensor payloads are ~150 bytes; anything near this limit is abuse.
const maxRequestBody = 64 * 1024

type responseCapture struct {
    gin.ResponseWriter
    body bytes.Buffer
}

func (w *responseCapture) Write(b []byte) (int, error) {
    w.body.Write(b)
    return w.ResponseWriter.Write(b)
}

// requestLogger logs one line per request with method, path, bodies,
// status, latency, and client IP. Headers are deliberately not logged so
// the X-API-Key value never reaches the logs.
func requestLogger() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()

        w := &responseCapture{ResponseWriter: c.Writer}
        c.Writer = w

        var reqBody []byte
        if c.Request.Body != nil && c.Request.ContentLength != 0 {
            var err error
            c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRequestBody)
            reqBody, err = io.ReadAll(c.Request.Body)
            if err != nil {
                c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": "request body too large"})
            } else {
                c.Request.Body = io.NopCloser(bytes.NewReader(reqBody))
            }
        }

        c.Next()

        msg := fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.RequestURI())
        if len(reqBody) > 0 {
            msg += " req=" + formatBody(reqBody)
        }
        msg += fmt.Sprintf(" -> %d in %s from %s", w.Status(), time.Since(start).Round(time.Millisecond), c.ClientIP())
        if w.body.Len() > 0 {
            msg += " resp=" + formatBody(w.body.Bytes())
        }
        log.Print(msg)
    }
}

// formatBody compacts JSON bodies onto one line and truncates long ones.
func formatBody(b []byte) string {
    var compact bytes.Buffer
    if err := json.Compact(&compact, b); err == nil {
        b = compact.Bytes()
    }
    if len(b) > maxLoggedBody {
        return fmt.Sprintf("%s...(%d bytes total)", b[:maxLoggedBody], len(b))
    }
    return string(b)
}

func apiKeyMiddleware(apiKey string) gin.HandlerFunc {
    return func(c *gin.Context) {
        key := c.GetHeader("X-API-Key")
        if subtle.ConstantTimeCompare([]byte(key), []byte(apiKey)) != 1 {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
            c.Abort()
            return
        }
        c.Next()
    }
}

// parseCoords reads lat/lon query params, writing a 400 response and
// returning ok=false if either is missing or invalid.
func parseCoords(c *gin.Context) (lat, lon float64, ok bool) {
    lat, err := strconv.ParseFloat(c.Query("lat"), 64)
    if err != nil || lat < -90 || lat > 90 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid latitude"})
        return 0, 0, false
    }

    lon, err = strconv.ParseFloat(c.Query("lon"), 64)
    if err != nil || lon < -180 || lon > 180 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid longitude"})
        return 0, 0, false
    }

    return lat, lon, true
}

type sensorReadingBody struct {
    SensorName  string   `json:"sensor_name" binding:"max=64"`
    Latitude    *float64 `json:"latitude" binding:"required"`
    Longitude   *float64 `json:"longitude" binding:"required"`
    Temperature *float64 `json:"temperature" binding:"required"`
    Humidity    *float64 `json:"humidity" binding:"required"`
}

func main() {
    apiKey := os.Getenv("API_KEY")
    if apiKey == "" {
        log.Fatal("API_KEY environment variable must be set")
    }

    dataFile := os.Getenv("SENSOR_DATA_FILE")
    if dataFile == "" {
        dataFile = "data/sensor_readings.jsonl"
    }
    store, err := sensor.NewStore(dataFile)
    if err != nil {
        log.Fatalf("failed to open sensor store: %v", err)
    }

    weatherClient := weather.NewWeatherClient()

    r := gin.New()
    r.Use(gin.Recovery(), requestLogger())

    r.GET("/api/v1/weather", apiKeyMiddleware(apiKey), func(c *gin.Context) {
        lat, lon, ok := parseCoords(c)
        if !ok {
            return
        }

        result, err := weatherClient.GetWeather(weather.WeatherRequest{
            Latitude:  lat,
            Longitude: lon,
        })
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }

        reading, found := store.LatestNear(lat, lon, sensorRadiusKm)
        if found && time.Since(time.Unix(reading.Timestamp, 0)) <= maxSensorAge {
            result.CurrentTemperature = int(math.Round(reading.Temperature))
            result.Humidity = int(math.Round(reading.Humidity))
        }

        c.JSON(http.StatusOK, result)
    })

    r.POST("/api/v1/sensor/readings", apiKeyMiddleware(apiKey), func(c *gin.Context) {
        var body sensorReadingBody
        if err := c.ShouldBindJSON(&body); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "latitude, longitude, temperature, and humidity are required"})
            return
        }
        if *body.Latitude < -90 || *body.Latitude > 90 {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid latitude"})
            return
        }
        if *body.Longitude < -180 || *body.Longitude > 180 {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid longitude"})
            return
        }

        reading, err := store.Add(sensor.Reading{
            SensorName:  body.SensorName,
            Latitude:    *body.Latitude,
            Longitude:   *body.Longitude,
            Temperature: *body.Temperature,
            Humidity:    *body.Humidity,
            Timestamp:   time.Now().Unix(),
        })
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save reading"})
            return
        }

        c.JSON(http.StatusCreated, reading)
    })

    r.GET("/api/v1/sensor/readings", apiKeyMiddleware(apiKey), func(c *gin.Context) {
        lat, lon, ok := parseCoords(c)
        if !ok {
            return
        }

        limit := 100
        if limitStr := c.Query("limit"); limitStr != "" {
            parsed, err := strconv.Atoi(limitStr)
            if err != nil || parsed < 1 || parsed > 10000 {
                c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
                return
            }
            limit = parsed
        }

        c.JSON(http.StatusOK, gin.H{
            "readings": store.HistoryNear(lat, lon, sensorRadiusKm, limit),
        })
    })

    r.Run(":6769")
}
