package ui

import (
	"ascii1090/internal/adsb"
	"ascii1090/internal/geo"
	"context"
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
)

// ViewMode represents the current view mode
type ViewMode int

const (
	ViewModeMap ViewMode = iota
	ViewModeDetail
)

// App is the main application controller
type App struct {
	screen      tcell.Screen
	tracker     *adsb.Tracker
	dump1090    *adsb.Dump1090Client
	mapView     *MapView
	listView    *ListView
	detailView  *DetailView
	currentView ViewMode
	quit        chan struct{}
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewApp creates a new application
func NewApp(tracker *adsb.Tracker, dump1090 *adsb.Dump1090Client, features map[geo.FeatureType][]*geo.Feature, radiusMiles float64, aspectRatio float64) (*App, error) {
	// Initialize tcell screen
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, fmt.Errorf("failed to create screen: %w", err)
	}

	if err := screen.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize screen: %w", err)
	}

	screen.SetStyle(tcell.StyleDefault)
	screen.Clear()

	width, height := screen.Size()

	mapView := NewMapView(width, height, features, radiusMiles, aspectRatio)

	// List view in lower-left corner
	listWidth := 30
	listHeight := 12
	listView := NewListView(0, height-listHeight, listWidth, listHeight)

	// Detail view in lower-left corner
	detailWidth := 50
	detailHeight := 15
	detailView := NewDetailView(0, height-detailHeight, detailWidth, detailHeight)

	ctx, cancel := context.WithCancel(context.Background())

	app := &App{
		screen:      screen,
		tracker:     tracker,
		dump1090:    dump1090,
		mapView:     mapView,
		listView:    listView,
		detailView:  detailView,
		currentView: ViewModeMap,
		quit:        make(chan struct{}),
		ctx:         ctx,
		cancel:      cancel,
	}

	return app, nil
}

// Run starts the application main loop
func (a *App) Run() error {
	defer a.cleanup()

	a.dump1090.Start()

	a.tracker.StartPruning(a.ctx, 10*time.Second)

	go a.readMessages()

	ticker := time.NewTicker(100 * time.Millisecond) // 10 FPS
	defer ticker.Stop()

	for {
		select {
		case <-a.quit:
			return nil

		case <-ticker.C:
			a.update()
			a.render()

		default:
			if a.screen.HasPendingEvent() {
				ev := a.screen.PollEvent()
				if !a.handleEvent(ev) {
					return nil // Quit requested
				}
			}
		}
	}
}

// readMessages reads aircraft updates from dump1090
func (a *App) readMessages() {
	for {
		select {
		case <-a.ctx.Done():
			return
		case ac := <-a.dump1090.ReadMessages():
			if ac != nil {
				a.tracker.Update(ac)
			}
		}
	}
}

// update updates the application state
func (a *App) update() {
	aircraft := a.tracker.GetAll()

	a.listView.Update(aircraft)

	a.mapView.SetCenterFromFirstAircraft(aircraft)

	if a.currentView == ViewModeDetail {
		selected := a.listView.GetSelected()
		a.detailView.SetAircraft(selected)
	}
}

// render renders the current view to the screen
func (a *App) render() {
	a.screen.Clear()

	aircraft := a.tracker.GetAll()
	selectedICAO := ""
	if selected := a.listView.GetSelected(); selected != nil {
		selectedICAO = selected.ICAO
	}

	// Always draw map
	a.mapView.Draw(a.screen, aircraft, selectedICAO)

	// Draw list or detail view depending on mode
	switch a.currentView {
	case ViewModeMap:
		a.listView.Draw(a.screen)
	case ViewModeDetail:
		a.detailView.Draw(a.screen)
	}

	a.screen.Show()
}

// handleEvent processes keyboard events
func (a *App) handleEvent(ev tcell.Event) bool {
	switch ev := ev.(type) {
	case *tcell.EventKey:
		switch ev.Key() {
		case tcell.KeyEscape:
			if a.currentView == ViewModeDetail {
				a.currentView = ViewModeMap
			} else {
				close(a.quit)
				return false
			}

		case tcell.KeyEnter:
			if a.currentView == ViewModeMap {
				a.currentView = ViewModeDetail
				selected := a.listView.GetSelected()
				a.detailView.SetAircraft(selected)
			}

		case tcell.KeyUp:
			if a.currentView == ViewModeMap {
				a.listView.SelectPrev()
				selected := a.listView.GetSelected()
				a.mapView.CenterOnAircraft(selected)
			}

		case tcell.KeyDown:
			if a.currentView == ViewModeMap {
				a.listView.SelectNext()
				selected := a.listView.GetSelected()
				a.mapView.CenterOnAircraft(selected)
			}

		case tcell.KeyRune:
			switch ev.Rune() {
			case 'q', 'Q':
				close(a.quit)
				return false

			case 'r', 'R':
				a.render()

			case '+', '=':
				a.mapView.ZoomIn()

			case '-', '_':
				a.mapView.ZoomOut()
			}
		}

	case *tcell.EventResize:
		a.handleResize()
	}

	return true
}

// handleResize handles terminal resize events
func (a *App) handleResize() {
	a.screen.Sync()
	width, height := a.screen.Size()

	a.mapView.UpdateDimensions(width, height)

	listWidth := 30
	listHeight := 12
	a.listView.UpdateDimensions(0, height-listHeight, listWidth, listHeight)

	detailWidth := 50
	detailHeight := 15
	a.detailView.UpdateDimensions(0, height-detailHeight, detailWidth, detailHeight)
}

// cleanup performs cleanup before exit
func (a *App) cleanup() {
	if a.cancel != nil {
		a.cancel()
	}

	if a.dump1090 != nil {
		a.dump1090.Close()
	}

	if a.screen != nil {
		a.screen.Fini()
	}
}
