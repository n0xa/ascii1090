# ascii1090

A terminal-based ADS-B aircraft tracker with ASCII map rendering. Display real-time aircraft positions on a map in your terminal using data from dump1090.

## Features

- **Real-time aircraft tracking** from dump1090
- **ASCII map rendering** with geographic features:
  - State borders (dark grey)
  - Highways (yellow)
  - Rivers (cyan)
  - Coastlines (dark blue)
  - Cities labeled
- **150-mile default radius view** starts off in central US, then jumps to first detected aircraft
- **8-direction aircraft symbols** based on heading: < ^ v > (cardinal) and ┌ ┐ └ ┘ (diagonal)
- **Scrollable aircraft list** (max 10 visible)
- **Detail view** with full aircraft information
- **Automatic map data download** from Natural Earth
- **Local or network dump1090** support

## Requirements

- Go 1.21 or later
- dump1090 installed and in PATH (for local mode)
  - OR access to a remote dump1090 instance (network mode)
- Terminal with Unicode support
- RTL-SDR dongle (if running dump1090 locally)

## Installation

```bash
# Clone or navigate to the project directory
cd ascii1090

# Build the project
go build 

# Run it
./ascii1090
```

## Usage

### Local Mode (spawn dump1090 automatically)

```bash
./ascii1090
```

This will:
1. Download Natural Earth map data (first run only, ~100MB)
2. Start dump1090 locally
3. Connect to dump1090's SBS output port (30003)
4. Display aircraft on the map

### Network Mode (connect to remote dump1090)

```bash
./ascii1090 -network 192.168.1.100:30003
```

### Command Line Options

- `-h` - Show help message
- `-network <host:port>` - Connect to remote dump1090 (default: start local dump1090)
- `-cache <dir>` - Cache directory for map data (default: `~/.ascii1090/data`)
- `-r <miles>` - Map radius in miles (default: 150)
- `-a <ratio>` - Character aspect ratio for font width adjustment (1.0-4.0, default: 2.0)
- `-H <level>` - Highway detail level, lower = fewer roads (1-10, default: 4)
- `-d <file>` - Enable debug logging to specified file (e.g., debug.log)

## Controls

### Map View (default)

- **Up/Down arrows** - Scroll through aircraft list
- **Enter** - Switch to detail view for selected aircraft
- **+** or **=** - Zoom in (decrease radius by 25%, min 10 miles)
- **-** or **_** - Zoom out (increase radius by 33%, max 1000 miles)
- **Q** or **ESC** - Quit application
- **R** - Force refresh

### Detail View

- **ESC** - Return to map view

## Aircraft List Format

```
(+) UAL123 FL450 500kts
( ) A12345 FL0   0kts
```

- `(+)` - Position coordinates are locked
- `( )` - No position lock yet
- **Flight number** or ICAO hex (7 chars)
- **FL###** - Flight level (altitude / 100)
- **###kts** - Ground speed in knots

## Detail View Information

- ICAO hex identifier
- Flight number (if available)
- Position (lat/lon)
- Altitude in feet and flight level
- Speed in knots
- Heading and ground track
- Vertical rate
- Time since last seen

## Map Features

- **State borders**: Dark grey lines `-`
- **Highways**: Yellow lines `=`
- **Rivers**: Cyan wavy lines `~`
- **Coastlines**: Dark blue lines `-`
- **Cities**: White text labels (no symbol)
- **Airports**: Orange `@` with airport code labels
- **Aircraft**: 8-direction symbols in green:
  - Cardinal: `^` (N), `>` (E), `v` (S), `<` (W)
  - Diagonal: `┐` (NE), `┘` (SE), `└` (SW), `┌` (NW)
- **Selected aircraft**: Bold/reversed aircraft symbol

Note: City labels are hidden when they overlap with airports to reduce clutter.

## Data Management

- Aircraft not seen for 60+ seconds are automatically removed
- Map data is downloaded once and cached locally
- Natural Earth 1:50m (medium detail) data used for geographic features
- Natural Earth 1:10m roads data for North American highways
- Initial download is larger (~50-100MB) but provides much better detail

## Troubleshooting

### "failed to start dump1090"

Make sure dump1090 is installed and in your PATH:
```bash
which dump1090
```

Alternatively, use network mode to connect to a remote instance.

### "failed to download map data"

Check your internet connection. Map data is downloaded from naturalearthdata.com on first run.

### No aircraft appearing

- Ensure your RTL-SDR dongle is connected
- Check that dump1090 is receiving data
- Verify you're in an area with ADS-B coverage
- In network mode, verify the host:port is correct

### Terminal too small

The application works best with at least 80x24 terminal size. Resize your terminal if the display looks compressed.

## Project Structure

```
ascii1090/
├── main.go              # Entry point
├── internal/
│   ├── adsb/            # Aircraft data and dump1090 client
│   ├── geo/             # Geographic data and projection
│   ├── render/          # Canvas and map rendering
│   ├── ui/              # TUI components
│   └── cache/           # Natural Earth data management
└── data/                # Cached map data
```

## Architecture

- **Aircraft Tracking**: Thread-safe tracker with automatic pruning
- **Map Rendering**: Bresenham line algorithm for efficient ASCII drawing
- **Coordinate Projection**: Equirectangular projection for regional views
- **TUI Framework**: tcell for terminal rendering and event handling
- **Data Format**: SBS/BaseStation format from dump1090 (port 30003)

## Dependencies

- [tcell](https://github.com/gdamore/tcell) - Terminal UI framework
- [go-shp](https://github.com/jonas-p/go-shp) - Shapefile parsing
- [Natural Earth](https://www.naturalearthdata.com/) - Map data (downloaded automatically)

## License

MIT License

## Future Enhancements

- Color-code aircraft by altitude
- Trail rendering (last N positions)
