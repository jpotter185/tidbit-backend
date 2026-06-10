# tidbit-backend

A lightweight Go REST API that proxies [Open-Meteo](https://open-meteo.com/) weather data into a clean, structured response. Built to serve the [Matrix Portal S3](https://www.adafruit.com/product/5778) LED weather display over a local network.

## Stack

- **Go** with [Gin](https://github.com/gin-gonic/gin)
- **Docker** for deployment
- Hosted on a local Ubuntu server

## Project Structure

```
tidbit-backend/
├── main.go
├── Dockerfile
└── internal/
    └── weather/
        ├── handler.go
        └── models.go
```

## Endpoints

### `GET /api/v1/weather`

Fetches current weather conditions from Open-Meteo for a given location.

**Query Parameters**

| Parameter | Type  | Description        |
|-----------|-------|--------------------|
| `lat`     | float | Latitude           |
| `lon`     | float | Longitude          |

**Example Request**

```bash
curl "http://localhost:6769/api/v1/weather?lat=39.9612&lon=-82.9988"
```

**Example Response**

```json
{
  "current_temperature": 72,
  "feels_like_temperature": 70,
  "is_day": 1,
  "weather_code": 2,
  "humidity": 55,
  "min_temp": 61,
  "max_temp": 78,
  "precip_probability": 10,
  "uv_index": 4,
  "tz_offset": -5
}
```

## Running Locally

```bash
go mod download
go run main.go
```

## Docker

**Build**

```bash
docker build -t tidbit-backend .
```

**Run**

```bash
docker run -d -p 6769:6769 -e PORT=6769 --name tidbit-backend tidbit-backend
```

**Logs**

```bash
docker logs tidbit-backend
```

## Deployment

The server is hosted at `192.168.1.201` on the local network. The Matrix Portal S3 hits the API directly over WiFi:

```
http://192.168.1.201:6769/api/v1/weather?lat=LATITUDE&lon=LONGITUDE
```