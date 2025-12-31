package geo

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
)

// AirportLoader loads airport data from OurAirports CSV
type AirportLoader struct {
	csvPath string
}

// NewAirportLoader creates a new airport loader
func NewAirportLoader(csvPath string) *AirportLoader {
	return &AirportLoader{
		csvPath: csvPath,
	}
}

// LoadAirports loads airports from the OurAirports CSV file
// Returns a slice of Feature objects representing airports
// Only loads medium_airport and large_airport types for reasonable density
func (a *AirportLoader) LoadAirports() ([]*Feature, error) {
	file, err := os.Open(a.csvPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open airports CSV: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Read header row to get column indices
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	colIndices := make(map[string]int)
	for i, col := range header {
		colIndices[col] = i
	}

	required := []string{"type", "name", "latitude_deg", "longitude_deg", "iata_code", "ident"}
	for _, col := range required {
		if _, ok := colIndices[col]; !ok {
			return nil, fmt.Errorf("missing required column: %s", col)
		}
	}

	var airports []*Feature

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		// Filter by type - only medium and large airports
		airportType := record[colIndices["type"]]
		if airportType != "medium_airport" && airportType != "large_airport" {
			continue
		}

		latStr := record[colIndices["latitude_deg"]]
		lonStr := record[colIndices["longitude_deg"]]

		lat, err := strconv.ParseFloat(latStr, 64)
		if err != nil {
			continue 
		}

		lon, err := strconv.ParseFloat(lonStr, 64)
		if err != nil {
			continue
		}

		name := record[colIndices["name"]]
		iataCode := record[colIndices["iata_code"]]
		ident := record[colIndices["ident"]]

		// Prefer IATA code for label (3-letter like DFW, LAX)
		// Fall back to ICAO ident if no IATA code
		label := name
		if iataCode != "" {
			label = iataCode
		} else if ident != "" {
			label = ident
		}

		airport := NewPointFeature(FeatureAirport, LatLon{Lat: lat, Lon: lon}, label)
		airport.Properties["full_name"] = name
		airport.Properties["type"] = airportType

		airports = append(airports, airport)
	}

	return airports, nil
}

// LoadAirportsInBounds loads only airports within the given geographic bounds
// This is more efficient when you only need airports in a specific region
func (a *AirportLoader) LoadAirportsInBounds(bounds *Bounds) ([]*Feature, error) {
	allAirports, err := a.LoadAirports()
	if err != nil {
		return nil, err
	}

	var filtered []*Feature
	for _, airport := range allAirports {
		if airport.Point != nil && bounds.Contains(airport.Point.Lat, airport.Point.Lon) {
			filtered = append(filtered, airport)
		}
	}

	return filtered, nil
}
