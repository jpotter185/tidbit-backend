package main

import (
	"net/http"
	"os"
	"strconv"

	"tidbit-backend/internal/weather"

	"github.com/gin-gonic/gin"
)

func apiKeyMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        key := c.GetHeader("X-API-Key")
        if key != os.Getenv("API_KEY") {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
            c.Abort()
            return
        }
        c.Next()
    }
}

func main() {
    weatherClient := weather.NewWeatherClient()

    r := gin.Default()

    r.GET("/api/v1/weather", apiKeyMiddleware(), func(c *gin.Context) {
        latStr := c.Query("lat")
        lonStr := c.Query("lon")

        lat, err := strconv.ParseFloat(latStr, 64)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid latitude"})
            return
        }

        lon, err := strconv.ParseFloat(lonStr, 64)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid longitude"})
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

        c.JSON(http.StatusOK, result)
    })

    
    r.Run(":6769")
}