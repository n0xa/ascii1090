package adsb

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Dump1090Client connects to a dump1090 instance and reads aircraft data
type Dump1090Client struct {
	conn        io.ReadCloser
	isLocalCLI  bool
	cmd         *exec.Cmd
	networkAddr string
	parser      *SBSParser
	msgChan     chan *Aircraft
	errChan     chan error
	done        chan struct{}
	closeOnce   sync.Once
}

// SBSParser parses SBS/BaseStation format messages
type SBSParser struct{}

// NewSBSParser creates a new SBS parser
func NewSBSParser() *SBSParser {
	return &SBSParser{}
}

// NewLocalClient spawns dump1090 CLI and connects to its SBS output
// dump1090 is launched with --net flag to enable network output on port 30003
func NewLocalClient() (*Dump1090Client, error) {
	// Spawn dump1090 with network output enabled
	cmd := exec.Command("dump1090", "--net", "--quiet")

	// Capture stderr to see any errors
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start dump1090: %w", err)
	}

	// Wait for dump1090 to initialize and open network port
	// Try to connect with retries
	var conn net.Conn
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		time.Sleep(500 * time.Millisecond)
		conn, err = net.Dial("tcp", "localhost:30003")
		if err == nil {
			break
		}
		if i == maxRetries-1 {
			// Read any error output from dump1090
			buf := make([]byte, 1024)
			n, _ := stderrPipe.Read(buf)
			errMsg := string(buf[:n])
			cmd.Process.Kill()
			return nil, fmt.Errorf("failed to connect to dump1090 SBS port after %d attempts: %w\nDump1090 stderr: %s", maxRetries, err, errMsg)
		}
	}

	return &Dump1090Client{
		conn:       conn,
		isLocalCLI: true,
		cmd:        cmd,
		parser:     NewSBSParser(),
		msgChan:    make(chan *Aircraft, 100),
		errChan:    make(chan error, 10),
		done:       make(chan struct{}),
	}, nil
}

// NewNetworkClient connects to a remote dump1090 instance via network
// addr should be in format "host:port", e.g., "192.168.1.100:30003"
func NewNetworkClient(addr string) (*Dump1090Client, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	return &Dump1090Client{
		conn:        conn,
		isLocalCLI:  false,
		networkAddr: addr,
		parser:      NewSBSParser(),
		msgChan:     make(chan *Aircraft, 100),
		errChan:     make(chan error, 10),
		done:        make(chan struct{}),
	}, nil
}

// Start begins reading messages from dump1090
func (c *Dump1090Client) Start() {
	go c.readLoop()
}

// ReadMessages returns a channel of parsed aircraft updates
func (c *Dump1090Client) ReadMessages() <-chan *Aircraft {
	return c.msgChan
}

// Errors returns a channel of errors encountered during parsing
func (c *Dump1090Client) Errors() <-chan error {
	return c.errChan
}

// Close closes the connection and stops dump1090 if running locally
func (c *Dump1090Client) Close() error {
	// Use sync.Once to ensure we only close once
	c.closeOnce.Do(func() {
		// Close the connection first to stop readLoop
		if c.conn != nil {
			c.conn.Close()
		}

		// Stop dump1090 process if running locally
		if c.isLocalCLI && c.cmd != nil && c.cmd.Process != nil {
			c.cmd.Process.Kill()
		}

		// Wait for readLoop to finish before closing channels
		<-c.done

		// Now safe to close channels
		close(c.msgChan)
		close(c.errChan)
	})
	return nil
}

// readLoop continuously reads and parses messages from dump1090
func (c *Dump1090Client) readLoop() {
	defer close(c.done) // Signal that readLoop is finished

	scanner := bufio.NewScanner(c.conn)
	for scanner.Scan() {
		line := scanner.Text()
		aircraft, err := c.parser.Parse(line)
		if err != nil {
			// Skip malformed lines silently
			continue
		}
		if aircraft != nil {
			select {
			case c.msgChan <- aircraft:
			case <-c.done:
				return // Exit if Close() was called
			}
		}
	}

	if err := scanner.Err(); err != nil {
		select {
		case c.errChan <- fmt.Errorf("error reading from dump1090: %w", err):
		case <-c.done:
			return // Exit if Close() was called
		}
	}
}

// Parse parses an SBS/BaseStation format message
// Format: MSG,transmission_type,session_id,aircraft_id,hex_ident,flight_id,date_generated,time_generated,date_logged,time_logged,callsign,altitude,ground_speed,track,lat,lon,vertical_rate,squawk,alert,emergency,spi,is_on_ground
// Example: MSG,3,,,A12345,,,2025/12/30,12:34:56.789,2025/12/30,12:34:56.789,,5000,,,37.7749,-122.4194,,,0,0,0,0
func (p *SBSParser) Parse(line string) (*Aircraft, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, nil
	}

	fields := strings.Split(line, ",")
	if len(fields) < 22 {
		return nil, fmt.Errorf("insufficient fields: %d", len(fields))
	}

	// Only process MSG messages
	if fields[0] != "MSG" {
		return nil, nil
	}

	// Extract ICAO hex identifier (field 4)
	icao := strings.TrimSpace(fields[4])
	if icao == "" {
		return nil, fmt.Errorf("missing ICAO")
	}

	aircraft := &Aircraft{
		ICAO:     icao,
		LastSeen: time.Now(),
	}

	// Callsign/Flight number (field 10)
	if fields[10] != "" {
		aircraft.FlightNumber = strings.TrimSpace(fields[10])
	}

	// Altitude in feet (field 11)
	if fields[11] != "" {
		if alt, err := strconv.Atoi(strings.TrimSpace(fields[11])); err == nil {
			aircraft.Altitude = alt
		}
	}

	// Ground speed in knots (field 12)
	if fields[12] != "" {
		if speed, err := strconv.Atoi(strings.TrimSpace(fields[12])); err == nil {
			aircraft.Speed = speed
		}
	}

	// Track/heading (field 13)
	if fields[13] != "" {
		if track, err := strconv.Atoi(strings.TrimSpace(fields[13])); err == nil {
			aircraft.Track = track
			aircraft.Heading = track // Use track as heading if not separately provided
		}
	}

	// Latitude (field 14)
	if fields[14] != "" {
		if lat, err := strconv.ParseFloat(strings.TrimSpace(fields[14]), 64); err == nil {
			aircraft.Latitude = &lat
		}
	}

	// Longitude (field 15)
	if fields[15] != "" {
		if lon, err := strconv.ParseFloat(strings.TrimSpace(fields[15]), 64); err == nil {
			aircraft.Longitude = &lon
		}
	}

	// Vertical rate in feet per minute (field 16)
	if fields[16] != "" {
		if vr, err := strconv.Atoi(strings.TrimSpace(fields[16])); err == nil {
			aircraft.VerticalRate = vr
		}
	}

	return aircraft, nil
}
