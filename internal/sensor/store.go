package sensor

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
)

// Reading is a single sensor report. Temperature unit is whatever the
// sensor sends (the LED display ecosystem uses fahrenheit); the server
// stores and returns it unconverted.
type Reading struct {
	ID          string  `json:"id"`
	SensorName  string  `json:"sensor_name"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
	Timestamp   int64   `json:"timestamp"`
}

// applyIdentity assigns a unique ID to the data point and fills in a
// default name for unnamed sensors.
func applyIdentity(r Reading) Reading {
	if r.ID == "" {
		r.ID = newID()
	}
	if r.SensorName == "" {
		r.SensorName = fmt.Sprintf("sensor@%.4f,%.4f", r.Latitude, r.Longitude)
	}
	return r
}

// newID returns a random 16-hex-char identifier.
func newID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Store keeps all readings in memory and appends each new one to a JSONL
// file so history survives restarts.
type Store struct {
	mu       sync.Mutex
	filePath string
	readings []Reading // chronological, oldest first
}

func NewStore(filePath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	s := &Store{filePath: filePath}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	f, err := os.Open(s.filePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to open readings file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var r Reading
		if err := json.Unmarshal(scanner.Bytes(), &r); err != nil {
			continue // skip corrupt lines rather than refusing to start
		}
		s.readings = append(s.readings, applyIdentity(r))
	}
	return scanner.Err()
}

// Add stores the reading and returns it with its identity fields filled in.
func (s *Store) Add(r Reading) (Reading, error) {
	r = applyIdentity(r)
	line, err := json.Marshal(r)
	if err != nil {
		return r, fmt.Errorf("failed to encode reading: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.OpenFile(s.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return r, fmt.Errorf("failed to open readings file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(line, '\n')); err != nil {
		return r, fmt.Errorf("failed to write reading: %w", err)
	}

	s.readings = append(s.readings, r)
	return r, nil
}

// LatestNear returns the most recent reading from any sensor within
// radiusKm of the given coordinates.
func (s *Store) LatestNear(lat, lon, radiusKm float64) (Reading, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := len(s.readings) - 1; i >= 0; i-- {
		r := s.readings[i]
		if distanceKm(lat, lon, r.Latitude, r.Longitude) <= radiusKm {
			return r, true
		}
	}
	return Reading{}, false
}

// HistoryNear returns up to limit readings within radiusKm of the given
// coordinates, most recent first.
func (s *Store) HistoryNear(lat, lon, radiusKm float64, limit int) []Reading {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := []Reading{}
	for i := len(s.readings) - 1; i >= 0 && len(out) < limit; i-- {
		r := s.readings[i]
		if distanceKm(lat, lon, r.Latitude, r.Longitude) <= radiusKm {
			out = append(out, r)
		}
	}
	return out
}

// distanceKm is the haversine great-circle distance between two points.
func distanceKm(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0
	toRad := func(deg float64) float64 { return deg * math.Pi / 180 }

	dLat := toRad(lat2 - lat1)
	dLon := toRad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLon/2)*math.Sin(dLon/2)
	return earthRadiusKm * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
