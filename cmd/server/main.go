package main

import (
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"
    "tidbit-backend/internal/weather"
)

func main() {
    weatherClient := weather.NewWeatherClient()

    r := gin.Default()

    r.GET("/api/v1/weather", func(c *gin.Context) {
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