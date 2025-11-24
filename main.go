package main

import (
	_ "embed"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/getlantern/systray"
)

// Embed

//go:embed assets/icon.ico
var trayIcon []byte

const (
	defaultInterval = 5 * time.Minute
)

// State-Variablen
var (
	configPath    string
	configMu      sync.RWMutex
	currentConfig Config

	ratesMu        sync.RWMutex
	nextUpdateMu   sync.RWMutex
	nextAutoUpdate time.Time
	rates          = map[string]float64{}

	triggeredMu   sync.Mutex
	lastTriggered = map[string]time.Time{}
	alarmCooldown = 5 * time.Minute

	openSettingsChan = make(chan struct{}, 1)
)

// Tray-Setup
func main() {
	runtime.LockOSThread()

	configPath = defaultConfigPath()

	if err := ensureConfig(); err != nil {
		fmt.Println("cannot create config:", err)
	} else {
		if err := loadConfig(); err != nil {
			fmt.Println("cannot load config:", err)
		}
	}

	go func() {
		for range openSettingsChan {
			go func() {
				runtime.LockOSThread()
				defer runtime.UnlockOSThread()
				openSettingsWindow()
			}()
		}
	}()

	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetIcon(trayIcon)

	systray.SetTitle("FX Tray")
	systray.SetTooltip("Loading FX rates...")

	mSettings := systray.AddMenuItem("Settings…", "Open settings window")
	mRefresh := systray.AddMenuItem("Refresh Rates", "Manually refresh FX rates")
	systray.AddSeparator()
	mLastUpdated := systray.AddMenuItem("Last Updated: N/A", "Last FX rates update time")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit application")

	// Menühandling
	go func() {
		for {
			select {
			case <-mSettings.ClickedCh:
				select {
				case openSettingsChan <- struct{}{}:
				default:
				}
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()

	// Refresh-Menü
	go func() {
		for {
			next := getNextAutoUpdate()
			if next.IsZero() {
				mRefresh.SetTitle("Refresh Rates")
			} else {
				remaining := time.Until(next)
				if remaining < 0 {
					remaining = 0
				}
				secs := int(remaining.Seconds())
				minutes := secs / 60
				seconds := secs % 60

				if secs == 0 {
					mRefresh.SetTitle("Refresh Rates (next: soon)")
				} else {
					mRefresh.SetTitle(
						fmt.Sprintf("Refresh Rates (next: %02d:%02d)", minutes, seconds),
					)
				}
			}

			time.Sleep(1 * time.Second)
		}
	}()

	// Manueller Refresh
	go func() {
		for range mRefresh.ClickedCh {
			if err := refreshRatesAndTooltip(); err != nil {
				fmt.Println("manual refresh:", err)
			}
			updateLastUpdated(mLastUpdated)
			setNextAutoUpdate()
		}
	}()

	// Anzeige "Last Updated"
	go func() {
		updateLastUpdated(mLastUpdated)

		for {
			time.Sleep(30 * time.Second)
			updateLastUpdated(mLastUpdated)
		}
	}()

	go updateLoop()
}

func onExit() {
}

// Auto-Update

func setNextAutoUpdate() {
	nextUpdateMu.Lock()
	defer nextUpdateMu.Unlock()
	nextAutoUpdate = time.Now().Add(defaultInterval)
}

func getNextAutoUpdate() time.Time {
	nextUpdateMu.RLock()
	defer nextUpdateMu.RUnlock()
	return nextAutoUpdate
}

func updateLastUpdated(m *systray.MenuItem) {
	now := time.Now().Format("15:04:05")
	m.SetTitle("Last Updated: " + now)
}
