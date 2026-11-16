package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gen2brain/beeep"
	"github.com/getlantern/systray"
)

type frankfurterRate struct {
	Date  string  `json:"date"`
	Base  string  `json:"base"`
	Quote string  `json:"quote"`
	Rate  float64 `json:"rate"`
}

func init() {
	beeep.AppName = "FX Tray App"
}

// Kurs-Update

// Config laden, Kurse aktualisieren
func updateLoop() {
	for {
		if err := loadConfig(); err != nil {
			fmt.Println("loadConfig:", err)
		}
		if err := refreshRatesAndTooltip(); err != nil {
			fmt.Println("refreshRates:", err)
		}

		setNextAutoUpdate()
		time.Sleep(defaultInterval)
	}
}

// Kurse holen, Tooltip aktualisieren, Alarme prüfen
func refreshRatesAndTooltip() error {
	configMu.RLock()
	cfg := currentConfig
	configMu.RUnlock()

	if len(cfg.Pairs) == 0 {
		systray.SetTooltip("No currency pairs configured")
		return nil
	}

	bases := map[string]struct{}{}
	for _, p := range cfg.Pairs {
		bases[strings.ToUpper(p.From)] = struct{}{}
	}

	tmpRates := map[string]float64{}

	for base := range bases {
		rr, err := fetchRates(base)
		if err != nil {
			return err
		}

		for _, p := range cfg.Pairs {
			if strings.ToUpper(p.From) != base {
				continue
			}
			key := pairKey(p.From, p.To)
			if rate, ok := rr.Rates[strings.ToUpper(p.To)]; ok {
				tmpRates[key] = rate
			}
		}
	}

	ratesMu.Lock()
	rates = tmpRates
	ratesMu.Unlock()

	var lines []string
	for _, p := range cfg.Pairs {
		key := pairKey(p.From, p.To)
		if rate, ok := tmpRates[key]; ok {
			lines = append(lines, fmt.Sprintf("%s: %.4f", key, rate))
		}
	}

	if len(lines) == 0 {
		systray.SetTooltip("No rates available")
	} else {
		systray.SetTooltip(strings.Join(lines, "\n"))
	}

	checkAlarms(cfg, tmpRates)

	return nil
}

// API Call

func fetchRates(base string) (*rateResponse, error) {
	base = strings.ToUpper(base)
	url := fmt.Sprintf("https://api.frankfurter.dev/v2/rates?base=%s", base)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fx api: status %d: %s", resp.StatusCode, string(body))
	}

	// v2 liefert ein flaches Array: eine Zeile pro Zielwährung
	var rows []frankfurterRate
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return nil, err
	}

	rr := &rateResponse{BaseCode: base, Rates: make(map[string]float64, len(rows))}
	for _, r := range rows {
		rr.Rates[strings.ToUpper(r.Quote)] = r.Rate
	}
	if len(rr.Rates) == 0 {
		return nil, fmt.Errorf("fx api: no rates for base %s", base)
	}

	return rr, nil
}

// Alarm

func checkAlarms(cfg Config, latest map[string]float64) {
	now := time.Now()

	for _, a := range cfg.Alarms {
		key := normalizeAlarmPair(a.Pair)
		rate, ok := latest[key]
		if !ok {
			continue
		}

		dir := strings.ToLower(strings.TrimSpace(a.Direction))
		trigKey := fmt.Sprintf("%s:%.4f:%s", key, a.Target, dir)

		shouldFire := false
		switch dir {
		case "above":
			if rate >= a.Target {
				shouldFire = true
			}
		case "below":
			if rate <= a.Target {
				shouldFire = true
			}
		default:
			continue
		}

		triggeredMu.Lock()
		lastTime, exists := lastTriggered[trigKey]
		canTrigger := !exists || now.Sub(lastTime) >= alarmCooldown

		if shouldFire && canTrigger {
			lastTriggered[trigKey] = now
			triggeredMu.Unlock()

			msg := fmt.Sprintf("%s is now %.4f (target %.4f %s)", key, rate, a.Target, dir)
			_ = beeep.Notify("FX Alarm", msg, "")
		} else {
			triggeredMu.Unlock()
		}
	}
}

// Helper

func normalizeAlarmPair(pair string) string {
	pair = strings.ToUpper(strings.TrimSpace(pair))
	pair = strings.ReplaceAll(pair, " ", "")

	if !strings.Contains(pair, "/") && len(pair) == 6 {
		pair = pair[:3] + "/" + pair[3:]
	}
	return pair
}

func pairKey(from, to string) string {
	return strings.ToUpper(strings.TrimSpace(from)) + "/" +
		strings.ToUpper(strings.TrimSpace(to))
}
