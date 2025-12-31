package adsb

import (
	"context"
	"sort"
	"sync"
	"time"
)

// Tracker manages a collection of aircraft with thread-safe access
type Tracker struct {
	aircraft map[string]*Aircraft // Keyed by ICAO hex
	mu       sync.RWMutex
	timeout  time.Duration
}

// NewTracker creates a new aircraft tracker
// timeout specifies how long before an aircraft is considered stale (default: 60s)
func NewTracker(timeout time.Duration) *Tracker {
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	return &Tracker{
		aircraft: make(map[string]*Aircraft),
		timeout:  timeout,
	}
}

// Update updates or adds an aircraft to the tracker
// If the aircraft already exists, it merges the new data (keeping non-zero values)
func (t *Tracker) Update(ac *Aircraft) {
	if ac == nil || ac.ICAO == "" {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	existing, exists := t.aircraft[ac.ICAO]
	if !exists {
		t.aircraft[ac.ICAO] = ac
		return
	}

	existing.LastSeen = ac.LastSeen

	if ac.FlightNumber != "" {
		existing.FlightNumber = ac.FlightNumber
	}

	if ac.Latitude != nil {
		existing.Latitude = ac.Latitude
	}

	if ac.Longitude != nil {
		existing.Longitude = ac.Longitude
	}

	if ac.Altitude != 0 {
		existing.Altitude = ac.Altitude
	}

	if ac.Speed != 0 {
		existing.Speed = ac.Speed
	}

	if ac.Heading != 0 {
		existing.Heading = ac.Heading
	}

	if ac.Track != 0 {
		existing.Track = ac.Track
	}

	if ac.VerticalRate != 0 {
		existing.VerticalRate = ac.VerticalRate
	}
}

// Get retrieves an aircraft by ICAO hex
func (t *Tracker) Get(icao string) (*Aircraft, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	ac, exists := t.aircraft[icao]
	return ac, exists
}

// GetAll returns all tracked aircraft sorted by ICAO
func (t *Tracker) GetAll() []*Aircraft {
	t.mu.RLock()
	defer t.mu.RUnlock()

	aircraft := make([]*Aircraft, 0, len(t.aircraft))
	for _, ac := range t.aircraft {
		aircraft = append(aircraft, ac)
	}

	// Sort by ICAO for consistent ordering
	sort.Slice(aircraft, func(i, j int) bool {
		return aircraft[i].ICAO < aircraft[j].ICAO
	})

	return aircraft
}

// GetWithPosition returns all aircraft that have valid position data
func (t *Tracker) GetWithPosition() []*Aircraft {
	all := t.GetAll()
	withPos := make([]*Aircraft, 0, len(all))

	for _, ac := range all {
		if ac.PositionLocked() {
			withPos = append(withPos, ac)
		}
	}

	return withPos
}

// Count returns the number of tracked aircraft
func (t *Tracker) Count() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.aircraft)
}

// PruneStale removes aircraft that haven't been seen in the timeout period
// Returns the number of aircraft removed
func (t *Tracker) PruneStale() int {
	t.mu.Lock()
	defer t.mu.Unlock()

	removed := 0
	for icao, ac := range t.aircraft {
		if ac.IsStale() {
			delete(t.aircraft, icao)
			removed++
		}
	}

	return removed
}

// Clear removes all aircraft from the tracker
func (t *Tracker) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.aircraft = make(map[string]*Aircraft)
}

// StartPruning starts a background goroutine that periodically prunes stale aircraft
// The goroutine runs until the context is cancelled
// pruneInterval specifies how often to check for stale aircraft (default: 10s)
func (t *Tracker) StartPruning(ctx context.Context, pruneInterval time.Duration) {
	if pruneInterval == 0 {
		pruneInterval = 10 * time.Second
	}

	go func() {
		ticker := time.NewTicker(pruneInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				t.PruneStale()
			}
		}
	}()
}

// GetFirstWithPosition returns the first aircraft with valid position data
// This is useful for determining the initial map center
func (t *Tracker) GetFirstWithPosition() *Aircraft {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Try to find aircraft with position in a deterministic order (sorted by ICAO)
	icaos := make([]string, 0, len(t.aircraft))
	for icao := range t.aircraft {
		icaos = append(icaos, icao)
	}
	sort.Strings(icaos)

	for _, icao := range icaos {
		ac := t.aircraft[icao]
		if ac.PositionLocked() {
			return ac
		}
	}

	return nil
}
