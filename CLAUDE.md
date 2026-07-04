# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A lightweight Go REST API that proxies [Open-Meteo](https://open-meteo.com/) weather data into a clean, structured response for a Matrix Portal S3 LED weather display. Single endpoint, single upstream dependency, no database.

## Commands

```bash
go mod download        # install dependencies
go run cmd/server/main.go   # run locally (listens on :6769)
go build -o /bin/tidbit-backend ./cmd/server   # build binary
go vet ./...            # static analysis
```

There are no tests in the repo yet. If adding Go tests, they'd run with `go test ./...`.

### Docker

```bash
docker build -t tidbit-backend .
docker run -d -p 6769:6769 -e PORT=6769 -e API_KEY=... --name tidbit-backend tidbit-backend
docker logs tidbit-backend
```

Local dev/prod-like run via compose (expects an external `app-network` and `API_KEY` in env):

```bash
docker compose up --build -d
```

## Architecture

- `cmd/server/main.go` — entrypoint. Sets up Gin, registers the `GET /api/v1/weather` route behind `apiKeyMiddleware`, parses `lat`/`lon` query params, and delegates to the weather client.
- `internal/weather/handler.go` — `WeatherClient` calls the Open-Meteo `/v1/forecast` endpoint with hardcoded query params (fahrenheit, mph, inches, `forecast_days=1`, `timezone=auto`), then `parseResponse` flattens Open-Meteo's nested current/daily/hourly arrays into a single flat `WeatherResponse`. Note: `parseResponse` extracts the current hour by slicing the ISO time string (`raw.Current.Time[11:13]`) to index into the hourly arrays for UV index and precip probability — hourly/daily arrays are otherwise assumed to align with `forecast_days=1`.
- `internal/weather/models.go` — two model sets: `openMeteo*` structs mirror the upstream API's JSON shape (unexported, decode-only), while `WeatherResponse` is the public flattened shape returned to clients.

## Auth

Requests must include `X-API-Key` matching the `API_KEY` env var (checked in `apiKeyMiddleware` in `main.go`). There's no key rotation or multiple-key support — it's a single shared secret for the LED display client.

## Deployment

Push to `main` triggers `.github/workflows/deploy.yml`, which SSHes (via `cloudflared access ssh`) into the home server at `ssh.tallyo.us`, `git pull`s, and runs `deploy.sh` (which does `docker compose down` + `docker compose up --build -d`). The server itself is reachable on the local network at `192.168.1.201:6769`.
