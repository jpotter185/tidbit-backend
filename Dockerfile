FROM golang:1.23-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/tidbit-backend ./cmd/server

FROM alpine:3.20

RUN adduser -D -H appuser
USER appuser

COPY --from=build /bin/tidbit-backend /bin/tidbit-backend

EXPOSE 6769
ENTRYPOINT ["/bin/tidbit-backend"]
