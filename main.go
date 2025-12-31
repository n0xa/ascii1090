package main

import (
	"ascii1090/internal/adsb"
	"ascii1090/internal/cache"
	"ascii1090/internal/debug"
	"ascii1090/internal/geo"
	"ascii1090/internal/ui"
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	// Parse command line flags
	help := flag.Bool("h", false, "Show help message")
	networkAddr := flag.String("network", "", "Connect to remote dump1090 (e.g., 192.168.1.100:30003)")
	cacheDir := flag.String("cache", "", "Cache directory for map data (default: ~/.ascii1090/data)")
	debugLog := flag.String("d", "", "Debug log file (e.g., debug.log)")
	radiusMiles := flag.Float64("r", 150.0, "Map radius in miles (default: 150)")
	aspectRatio := flag.Float64("a", 2.0, "Character aspect ratio - adjust for font width (1.0-4.0, default: 2.0)")
	highwayDetail := flag.Int("H", 4, "Highway detail level - lower shows fewer roads (1-10, default: 4)")
	flag.Parse()

	// Show help if requested
	if *help {
		fmt.Println("ascii1090 - Terminal-based ADS-B aircraft tracker")
		fmt.Println("\nUsage: ascii1090 [options]")
		fmt.Println("\nOptions:")
		flag.PrintDefaults()
		os.Exit(0)
	}

	// Validate aspect ratio
	if *aspectRatio < 1.0 || *aspectRatio > 4.0 {
		fmt.Fprintf(os.Stderr, "Error: Aspect ratio must be between 1.0 and 4.0\n")
		os.Exit(1)
	}

	// Validate highway detail level
	if *highwayDetail < 1 || *highwayDetail > 10 {
		fmt.Fprintf(os.Stderr, "Error: Highway detail level must be between 1 and 10\n")
		os.Exit(1)
	}

	// Set up debug logging if requested
	if *debugLog != "" {
		logFile, err := os.Create(*debugLog)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create debug log: %v\n", err)
		} else {
			defer logFile.Close()
			debug.SetOutput(logFile)
			debug.Log("ascii1090 debug log started")
			fmt.Printf("Debug logging enabled: %s\n", *debugLog)
		}
	}

	// Initialize cache manager
	fmt.Println("Initializing map data cache...")
	cacheManager, err := cache.NewManager(*cacheDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to initialize cache: %v\n", err)
		os.Exit(1)
	}

	// Ensure Natural Earth data is available
	fmt.Println("Checking Natural Earth data...")
	if err := cacheManager.EnsureData(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to download map data: %v\n", err)
		os.Exit(1)
	}

	// Load shapefiles
	fmt.Println("Loading geographic features...")
	loader := geo.NewShapefileLoader(cacheManager.GetCacheDir())
	features, err := loader.LoadAll(*highwayDetail)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load shapefiles: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Loaded %d feature types\n", len(features))

	// Initialize dump1090 client
	var dump1090Client *adsb.Dump1090Client
	if *networkAddr != "" {
		fmt.Printf("Connecting to dump1090 at %s...\n", *networkAddr)
		dump1090Client, err = adsb.NewNetworkClient(*networkAddr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to connect to dump1090: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println("Starting local dump1090...")
		dump1090Client, err = adsb.NewLocalClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to start dump1090: %v\n", err)
			fmt.Fprintf(os.Stderr, "Hint: Make sure dump1090 is installed and in your PATH\n")
			fmt.Fprintf(os.Stderr, "Or use -network flag to connect to a remote instance\n")
			os.Exit(1)
		}
	}
	defer dump1090Client.Close()

	// Initialize aircraft tracker
	tracker := adsb.NewTracker(60 * time.Second)

	// Create and run application
	fmt.Printf("Starting ascii1090 (radius: %.0f miles, aspect: %.1f)...\n", *radiusMiles, *aspectRatio)
	app, err := ui.NewApp(tracker, dump1090Client, features, *radiusMiles, *aspectRatio)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create application: %v\n", err)
		os.Exit(1)
	}

	// Run with panic recovery to ensure terminal is always restored
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "\nPanic: %v\n", r)
			}
		}()

		if err := app.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
	}()

	fmt.Println("\nGoodbye!")
}
