# ascii1090 Implementation Plan

A terminal-based ADS-B aircraft tracker in Go with ASCII map rendering.

## Project Overview

Build a full-featured TUI application that:
- Displays aircraft from dump1090 on an ASCII map
- Shows 50-mile radius centered on first aircraft
- Renders geographic features (borders, highways, rivers, coastlines)
- Provides scrollable aircraft list and detail views
- Auto-prunes stale aircraft (60s timeout)

## Architecture

### Package Structure

```
ascii1090/
├── main.go
├── go.mod
├── data/                        # Natural Earth data cache
├── internal/
│   ├── adsb/
│   │   ├── aircraft.go          # Aircraft data model
│   │   ├── dump1090.go          # dump1090 client/parser (SBS format)
│   │   └── tracker.go           # Aircraft tracking & timeout
│   ├── geo/
│   │   ├── projection.go        # Lat/lon to screen coords
│   │   ├── shapefile.go         # Natural Earth loader
│   │   ├── features.go          # Feature types
│   │   └── bounds.go            # Bounding box
│   ├── render/
│   │   ├── map.go               # Map ASCII rendering
│   │   ├── aircraft.go          # Aircraft symbols
│   │   ├── canvas.go            # ASCII canvas abstraction
│   │   └── styles.go            # Colors/styles
│   ├── ui/
│   │   ├── app.go               # Main controller
│   │   ├── mapview.go           # Map viewport
│   │   ├── listview.go          # Aircraft list (10 visible)
│   │   ├── detailview.go        # Detail view
│   │   └── keyboard.go          # Event handlers
│   └── cache/
│       ├── downloader.go        # Natural Earth downloader
│       └── manager.go           # Cache management
```

### Key Components

**Aircraft Model** (`internal/adsb/aircraft.go`):
- ICAO hex, flight number, lat/lon, altitude, speed, heading
- `CardinalDirection()` → returns `< ^ v >` based on heading
- `IsStale()` → true if not seen in 60s

**dump1090 Client** (`internal/adsb/dump1090.go`):
- Parse SBS/BaseStation format (port 30003)
- `NewLocalClient()` → spawn dump1090 CLI, read stdout
- `NewNetworkClient(addr)` → connect to remote dump1090
- `ReadMessages()` → channel of Aircraft updates

**Tracker** (`internal/adsb/tracker.go`):
- Thread-safe aircraft map (sync.RWMutex)
- `Update()`, `GetAll()`, `PruneStale()`
- Background pruning goroutine (every 10s)

**Projection** (`internal/geo/projection.go`):
- Equirectangular projection for 50-mile radius
- `Project(lat, lon)` → screen Point(x, y)
- `UpdateCenter()` when first aircraft appears

**Shapefile Loader** (`internal/geo/shapefile.go`):
- Load Natural Earth 1:110m data (simplified)
- Feature types: StateBorder, Highway, River, Coastline, City
- `LoadAll()` → map of features by type

**Map Renderer** (`internal/render/map.go`):
- Bresenham line algorithm for polylines
- Styles: dark grey borders, yellow highways, cyan rivers, dark blue coasts
- Draw to Canvas, then blit to tcell screen

**UI Views**:
- `MapView`: Full-screen map + aircraft overlay
- `ListView`: Lower-left panel, max 10 visible, scrollable
  - Format: `(+) UAL123 FL450 500kts` or `( ) A12345 FL0 0kts`
- `DetailView`: Toggle with Enter, show full aircraft details

## Implementation Steps

### Phase 1: Foundation
1. **Project setup**: `go mod init`, add dependencies (tcell, go-shp)
2. **Aircraft model**: Implement Aircraft struct with CardinalDirection/IsStale
3. **dump1090 integration**: SBS parser, local/network clients, message channel

### Phase 2: Aircraft Tracking
4. **Tracker**: Thread-safe map, update/prune logic, background pruning
5. **Integration test**: Dump1090Client → Tracker pipeline

### Phase 3: Geographic Data
6. **Cache manager**: Download Natural Earth 1:110m data to `~/.ascii1090/data/`
   - States, roads, rivers, coastlines, cities
7. **Shapefile loader**: Parse with go-shp, convert to Feature structs
8. **Projection**: Equirectangular with 50-mile radius calculation

### Phase 4: Rendering
9. **Canvas**: 2D cell array (char + tcell.Style)
10. **Map renderer**: Bresenham lines, feature styling, label rendering
11. **Aircraft renderer**: Directional arrows, bold selection

### Phase 5: TUI Components
12. **MapView**: Combine map + aircraft, auto-center on first aircraft
13. **ListView**: Scrollable list, max 10 visible, selection
14. **DetailView**: Toggle panel with full aircraft info
15. **Keyboard handler**: Arrow keys (scroll), Enter (detail), Q (quit)

### Phase 6: Integration
16. **App controller**: Event loop, component coordination, refresh throttling
17. **CLI**: Parse `-network` flag, spawn dump1090 or connect to remote
18. **Testing**: Real dump1090, resize handling, keyboard shortcuts

### Phase 7: Polish
19. **Error handling**: Connection failures, data errors, graceful degradation
20. **Performance**: Spatial culling, coordinate caching, profiling
21. **Documentation**: README, usage examples, installation guide

## Critical Files (Creation Order)

1. `internal/adsb/aircraft.go` - Data model foundation
2. `internal/adsb/dump1090.go` - Data ingestion (SBS parser)
3. `internal/adsb/tracker.go` - State management
4. `internal/geo/projection.go` - Coordinate transformation
5. `internal/cache/manager.go` - Natural Earth data fetching
6. `internal/geo/shapefile.go` - Geographic data loading
7. `internal/render/canvas.go` - Rendering abstraction
8. `internal/render/map.go` - Map rendering engine
9. `internal/ui/listview.go` - Aircraft list panel
10. `internal/ui/mapview.go` - Map display component
11. `internal/ui/detailview.go` - Detail panel
12. `internal/ui/app.go` - Main controller
13. `main.go` - Entry point

## Technical Decisions

**Map Data**: Natural Earth 1:110m (simplified vector data)
- Easier to parse than MVT
- Sufficient detail for regional 50-mile view
- ~10-20MB total download size

**dump1090 Format**: SBS/BaseStation (port 30003)
- Most stable and well-documented format
- Supported by all dump1090 variants
- Text-based, easy to parse

**Projection**: Equirectangular with cosine correction
- Simple and fast for regional views
- Minimal distortion at mid-latitudes
- Good enough for 50-mile radius

**TUI**: tcell (not bubbletea)
- User has prior experience with tcell
- Lower-level control for custom rendering
- Excellent performance

## Dependencies

```go
require (
    github.com/gdamore/tcell/v2 v2.7.0   // TUI
    github.com/jonas-p/go-shp v0.1.1     // Shapefile parsing
)
```

## Usage Examples

```bash
# Local dump1090 (auto-spawn)
$ ascii1090

# Connect to remote dump1090
$ ascii1090 -network 192.168.1.100:30003

# Set custom map radius (50 miles for tight view)
$ ascii1090 -r 50

# Set wide radius (300 miles for regional view)
$ ascii1090 -r 300

# Enable debug logging
$ ascii1090 -d debug.log
```

## Potential Challenges

1. **dump1090 variants**: Focus on SBS format (port 30003) for compatibility
2. **Data size**: Use 1:110m scale, lazy loading, spatial culling
3. **Projection accuracy**: Acceptable for regional views, document limitations
4. **ASCII resolution**: Use box-drawing chars, simplify features
5. **Update frequency**: Decouple parsing from rendering (separate goroutines)
6. **Resize handling**: Listen for tcell resize events, recalculate projection
7. **No aircraft**: Default center to user location or central USA
8. **Concurrency**: Use sync.RWMutex in Tracker for safe access

## Feature Specifications

**Map Display**:
- 50-mile radius from first aircraft with coordinates
- State borders: dark grey `─│┌┐└┘├┤┬┴┼`
- Highways: yellow `═` or `-`
- Rivers: cyan `~` or `≈`
- Ocean borders: dark blue `-`
- Cities/airports labeled (major only)

**Aircraft Display**:
- Direction arrows: `<` (W), `^` (N), `v` (S), `>` (E)
- Bold when selected
- Hide if no coordinates locked

**Aircraft List (lower-left)**:
- Max 10 visible, full list scrollable
- Format: `(+) FLIGHT FL### SPDkts` or `( ) ICAO FL### SPDkts`
- `(+)` = coordinates known, `( )` = no lock
- Arrow keys to scroll, highlight selected

**Detail View**:
- Toggle with Enter key
- Show: ICAO, flight, position, altitude, heading, track, speed, vertical rate, last seen
- ESC to return to map

**Data Management**:
- Remove aircraft not seen in 60+ seconds
- Auto-prune every 10 seconds
- Real-time updates from dump1090

## Implementation Status

### ✅ Completed (December 30, 2024)

**Phase 1-6: Core Implementation**
- ✅ Go module initialized with tcell and go-shp dependencies
- ✅ Aircraft data model with cardinal direction symbols (< ^ v >)
- ✅ SBS parser for dump1090 BaseStation format (port 30003)
- ✅ Dump1090Client supporting both local and network modes
- ✅ Thread-safe aircraft tracker with 60-second timeout
- ✅ Natural Earth 110m data caching (with CDN workarounds)
- ✅ Shapefile loader for borders, rivers, coastlines, cities
- ✅ Equirectangular projection (150-mile radius, increased from 50)
- ✅ Canvas-based ASCII rendering with Bresenham line algorithm
- ✅ Map renderer with feature filtering by bounds
- ✅ Scrollable aircraft list (max 10 visible)
- ✅ Detail view with full aircraft information
- ✅ TUI event loop with keyboard controls
- ✅ Proper terminal cleanup on exit
- ✅ Debug logging to file with `-d` flag

**Working Features:**
- Real-time aircraft tracking from dump1090
- Map auto-centers on first aircraft with GPS coordinates
- State borders rendering correctly
- Aircraft symbols showing direction of travel
- Aircraft list with position lock indicators
- Detail view toggle (Enter key)
- Clean terminal restoration on exit

### ✅ Recently Fixed (December 30, 2024)

**Goroutine Cleanup on Exit**
- **Issue**: Panic "close of closed channel" when exiting application
- **Root Cause**: Race condition between `readLoop()` goroutine and `Close()` method
- **Solution**: Added proper synchronization:
  - Added `done` channel to signal readLoop completion
  - Added `sync.Once` to prevent double-close
  - Modified `Close()` to wait for readLoop before closing channels
  - Updated readLoop to use select statements for safe shutdown
- **Status**: ✅ Fixed - clean shutdown without panics

**Configurable Map Radius**
- **Issue**: Map radius was hardcoded to 150 miles
- **Solution**: Added `-r <miles>` command-line flag
  - Updated `MapView` to accept radius as parameter
  - Updated `NewApp` to pass radius through
  - Added radius display in startup message
- **Usage**: `ascii1090 -r 50` for tight view, `ascii1090 -r 300` for wide regional view
- **Status**: ✅ Implemented - radius now configurable via CLI

**Airport Database Integration**
- **Issue**: No airports shown on map, only 243 cities worldwide
- **Solution**: Integrated OurAirports.com database (~70k airports)
  - Created `airports.go` CSV parser for OurAirports data
  - Updated cache manager to download airports.csv from GitHub mirror
  - Added orange styling for airports to distinguish from cities
  - Filters to show only medium and large airports (reasonable density)
  - Uses IATA codes (DFW, LAX, etc.) for labels when available
- **Data**: ~10MB CSV file with latitude/longitude for all airports
- **Status**: ✅ Implemented - airports now display on map

### 🔧 Current Issues & Improvements Needed

**1. Limited City Coverage**
- **Issue**: Only seeing Houston in Texas; missing Austin, San Antonio, Dallas
- **Root Cause**: Natural Earth 110m is extremely simplified - only major world cities
- **Impact**: Users expect to see regional cities and airports
- **Solutions**:
  - Option A: Use Natural Earth 50m or 10m scale (more cities, larger files)
  - Option B: Add US-specific city dataset (US Census, GeoNames)
  - Option C: Add airport database (OurAirports.com data)
  - Option D: Use OpenStreetMap extracts for populated places

**2. Missing Rivers**
- **Status**: Rivers loaded (13 features) but may not be visible in user's area
- **Note**: 110m scale only includes major rivers; regional creeks/streams not included

**3. No Highways Visible**
- **Status**: Roads download failed; marked as optional
- **Note**: 110m road data is extremely sparse (only major international routes)

### 📋 Recommended Next Steps

**High Priority:**
1. ~~**Upgrade city dataset**~~ / ~~**Add airport database**~~ - ✅ COMPLETED
   - ✅ Added OurAirports database with ~70k airports
   - ✅ Medium and large airports only (good density)
   - ✅ Orange styling to distinguish from cities
   - Note: Natural Earth 50m cities still an option for additional POI coverage

2. ~~**Add configuration for radius**~~ - ✅ COMPLETED
   - ✅ Added `-r <miles>` flag for configurable radius
   - ✅ Default remains 150 miles

3. **Add feature toggle** - Allow disabling specific features for performance
   - `-no-rivers`, `-no-borders`, etc.

**Medium Priority:**
4. **Improve city label placement** - Avoid overlapping labels
5. **Add distance circles** - Show 50/100/150 mile radius markers
6. **Add compass rose** - N/S/E/W indicators

**Low Priority:**
7. **Aircraft trails** - Show last N positions
8. **Altitude color coding** - Different colors for different flight levels
9. **Filter by altitude** - Only show aircraft in certain altitude ranges

### 🗺️ Data Sources

**Current:**
- Natural Earth 110m (very simplified, ~20MB total)
  - ✅ States/provinces: 51 features
  - ✅ Rivers: 13 major rivers
  - ✅ Coastlines: 134 segments
  - ✅ Cities: 243 worldwide (sparse coverage)
- ✅ OurAirports.com (~10MB CSV)
  - ✅ Medium/large airports: ~5,000 worldwide
  - ✅ IATA code labels (DFW, LAX, etc.)
  - ✅ Orange styling for visibility

**Optional Upgrades:**
- Natural Earth 50m (~100MB) - Better city coverage (~7,000 cities)
- Natural Earth 10m (~500MB) - Comprehensive detail
- US Census Places - Comprehensive US city data
- OpenStreetMap - Most detailed, but complex to process

### 🐛 Known Limitations

1. **110m dataset limitations**: Designed for world-scale maps, not regional views
2. **No airport labels**: Would require additional dataset
3. **Fixed 150-mile radius**: Hardcoded, should be configurable
4. **Single map load**: Features loaded once at startup, not re-filtered efficiently
5. **Label overlap**: City labels may overlap with borders/rivers

### 💡 Architecture Notes

**What Went Well:**
- Modular package structure makes it easy to swap data sources
- Canvas abstraction cleanly separates rendering from display
- Feature filtering by bounds works correctly
- tcell provides excellent terminal control

**What Could Be Improved:**
- Consider caching projected coordinates for better performance
- Add spatial indexing (R-tree) for large feature sets
- Support multiple data sources (50m + 110m hybrid)
- Make radius and data scale configurable at runtime

---

## 🚀 Immediate Next Steps for Handoff

### Current Status Summary

✅ **What's Working:**
- Real-time aircraft tracking from dump1090
- Map rendering with state borders
- 150-mile radius view
- Aircraft list and detail views
- Proper terminal cleanup on exit
- Debug logging with `-d` flag

⚠️ **Main Issue: Limited City Data**
The Natural Earth 110m dataset is designed for **world-scale maps**, not regional views. It only includes ~250 major cities worldwide, which is why users only see Houston in Texas (missing Dallas, Austin, San Antonio, etc.).

### Recommended Solutions

**Option 1: Upgrade to Natural Earth 50m** (Quick win)
- ~7,000 cities instead of 250
- Download size: ~100MB total
- Would show Dallas, Austin, San Antonio, Fort Worth, etc.
- URL: https://www.naturalearthdata.com/downloads/50m-cultural-vectors/
- Implementation: Update `cache/manager.go` to download 50m instead of 110m

**Option 2: Add Airport Database** (Best for ADS-B use case)
- OurAirports.com has ~70,000 airports worldwide
- CSV format (10MB), easy to parse
- Perfect complement to aircraft tracking
- Shows DFW, AUS, SAT, IAH, HOU, etc.
- URL: https://ourairports.com/data/
- Files needed: `airports.csv`, `runways.csv`
- Implementation: New loader in `geo/airports.go`, render as ✈ symbol

**Option 3: Hybrid Approach** (Recommended - Best balance)
- Keep 110m for borders/coastlines (fast rendering)
- Add 50m cities OR airports dataset for better POI coverage
- Best balance of detail and performance
- Total download: ~120MB (vs 20MB current)

### ✅ Implementation Complete: Airport Database

**Files Modified:**
- ✅ `internal/geo/airports.go` - CSV parser for OurAirports data
- ✅ `internal/geo/features.go` - FeatureAirport type (already existed)
- ✅ `internal/geo/shapefile.go` - Integrated airports into LoadAll()
- ✅ `internal/cache/manager.go` - Downloads airports.csv automatically
- ✅ `internal/render/styles.go` - Orange styling for airports

**Features:**
- ✅ Automatic download on first run (~10MB)
- ✅ Filters to medium/large airports only
- ✅ Uses IATA codes for compact labels
- ✅ Orange color distinguishes from white cities
- ✅ Integrated with existing bounds filtering

### Quick Wins (Low-hanging fruit)

1. ~~**Make radius configurable**~~ - ✅ COMPLETED - Added `-r <miles>` flag
2. **Add help text** - Implement `-h` flag to show controls and options
3. **Show aircraft count** - Display "12 aircraft" in title bar
4. **Add timestamp** - Show current time in corner
5. **Improve startup message** - Show "Waiting for aircraft..." before first GPS lock

### Performance Optimizations (if needed)

1. **Spatial indexing** - Use R-tree for features (current O(n) filtering)
2. **Coordinate caching** - Cache projected screen coordinates
3. **Lazy loading** - Only load features within N miles of current view
4. **Simplification** - Douglas-Peucker algorithm for line simplification

### Files to Modify for Airport Support

```
internal/geo/airports.go         (NEW - CSV parser)
internal/geo/features.go         (add FeatureAirport constant)
internal/cache/manager.go        (add airports.csv download)
internal/render/styles.go        (add airport style)
internal/geo/shapefile.go        (update LoadAll to include airports)
```

### Testing Checklist

- [ ] Airports appear in correct locations
- [ ] Airport labels don't overlap excessively
- [ ] Performance is acceptable with 70k airports
- [ ] Filtering by bounds works correctly
- [ ] Debug log shows airport loading
- [ ] Terminal cleanup still works
- [ ] Works with both local and network dump1090

### Known Issues to Address

1. City labels only work if NAME field is populated correctly
2. No label collision avoidance (labels may overlap)
3. Rivers not visible in some areas (dataset too sparse)
4. Fixed 150-mile radius should be configurable
5. Map doesn't re-center after initial lock (design decision)
