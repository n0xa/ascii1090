package cache

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Manager handles downloading and caching Natural Earth data
type Manager struct {
	cacheDir string
}

// DataFile represents a Natural Earth dataset to download
type DataFile struct {
	Name     string // Friendly name
	URL      string // Download URL
	Base     string // Base filename (without extension)
	Optional bool   // If true, failure to download won't stop the app
}

// Natural Earth datasets - using 1:50m (medium detail) for most features
var NaturalEarthFiles = []DataFile{
	{
		Name:     "States/Provinces",
		URL:      "https://naciscdn.org/naturalearth/50m/cultural/ne_50m_admin_1_states_provinces.zip",
		Base:     "ne_50m_admin_1_states_provinces",
		Optional: true, // Optional - app can work without it
	},
	{
		Name:     "Rivers",
		URL:      "https://naciscdn.org/naturalearth/50m/physical/ne_50m_rivers_lake_centerlines.zip",
		Base:     "ne_50m_rivers_lake_centerlines",
		Optional: true,
	},
	{
		Name:     "Coastlines",
		URL:      "https://naciscdn.org/naturalearth/50m/physical/ne_50m_coastline.zip",
		Base:     "ne_50m_coastline",
		Optional: true,
	},
	{
		Name:     "Populated Places",
		URL:      "https://naciscdn.org/naturalearth/50m/cultural/ne_50m_populated_places.zip",
		Base:     "ne_50m_populated_places",
		Optional: true,
	},
	{
		Name:     "Roads (North America)",
		URL:      "https://naciscdn.org/naturalearth/10m/cultural/ne_10m_roads_north_america.zip",
		Base:     "ne_10m_roads_north_america",
		Optional: true,
	},
}

// NewManager creates a new cache manager
// If cacheDir is empty, uses ~/.ascii1090/data
func NewManager(cacheDir string) (*Manager, error) {
	if cacheDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		cacheDir = filepath.Join(home, ".ascii1090", "data")
	}

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &Manager{
		cacheDir: cacheDir,
	}, nil
}

// EnsureData ensures all required Natural Earth data is available
// Downloads missing files automatically
// Optional files that fail to download will be skipped with a warning
func (m *Manager) EnsureData() error {
	for _, file := range NaturalEarthFiles {
		if err := m.ensureFile(file); err != nil {
			if file.Optional {
				fmt.Printf("Warning: Skipping %s (optional): %v\n", file.Name, err)
				continue
			}
			return fmt.Errorf("failed to ensure %s: %w", file.Name, err)
		}
	}

	// Download airport database (optional)
	if err := m.EnsureAirportData(); err != nil {
		fmt.Printf("Warning: Failed to download airports (optional): %v\n", err)
	}

	return nil
}

// ensureFile checks if a data file exists, downloads if needed
func (m *Manager) ensureFile(file DataFile) error {
	shpPath := filepath.Join(m.cacheDir, file.Base+".shp")
	if _, err := os.Stat(shpPath); err == nil {
		return nil
	}

	fmt.Printf("Downloading %s...\n", file.Name)

	client := &http.Client{}
	req, err := http.NewRequest("GET", file.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ascii1090/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s (URL: %s)", resp.Status, file.URL)
	}

	tmpFile, err := os.CreateTemp("", "ne_*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return fmt.Errorf("failed to save download: %w", err)
	}

	tmpFile.Close()

	if err := m.extractZip(tmpFile.Name(), m.cacheDir); err != nil {
		return fmt.Errorf("failed to extract: %w", err)
	}

	fmt.Printf("Downloaded and extracted %s\n", file.Name)
	return nil
}

func (m *Manager) extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if f.FileInfo().IsDir() || strings.HasPrefix(filepath.Base(f.Name), ".") {
			continue
		}

		destPath := filepath.Join(destDir, filepath.Base(f.Name))
		rc, err := f.Open()

		if err != nil {
			return err
		}

		outFile, err := os.Create(destPath)
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) GetDataPath(base string) string {
	return filepath.Join(m.cacheDir, base+".shp")
}

func (m *Manager) GetCacheDir() string {
	return m.cacheDir
}

// EnsureAirportData downloads the OurAirports CSV if not already cached
func (m *Manager) EnsureAirportData() error {
	csvPath := filepath.Join(m.cacheDir, "airports.csv")

	if _, err := os.Stat(csvPath); err == nil {
		return nil
	}

	fmt.Println("Downloading airport database from OurAirports...")

	url := "https://davidmegginson.github.io/ourairports-data/airports.csv"

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ascii1090/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download airports: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	outFile, err := os.Create(csvPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, resp.Body); err != nil {
		return fmt.Errorf("failed to save airports CSV: %w", err)
	}

	fmt.Println("Downloaded airport database successfully")
	return nil
}

// GetAirportCSVPath returns the path to the airports CSV file
func (m *Manager) GetAirportCSVPath() string {
	return filepath.Join(m.cacheDir, "airports.csv")
}
