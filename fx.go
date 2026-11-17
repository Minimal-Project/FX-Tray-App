package main

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/gen2brain/beeep"
	"github.com/getlantern/systray"
)

func init() {
	beeep.AppName = "FX Tray App"
}

const maxTooltipLen = 127

func updateLoop() {
	for {
		if err := loadConfig(); err != nil {
			fmt.Println("loadConfig:", err)
		}
		if err := refreshRatesAndTooltip(); err != nil {
			fmt.Println("refreshRates:", err)
		}

		interval := currentInterval()
		setNextAutoUpdate(interval)

		select {
		case <-time.After(interval):
		case <-refreshNowChan:
		}
	}
}

func requestRefresh() {
	select {
	case refreshNowChan <- struct{}{}:
	default:
	}
}

func currentInterval() time.Duration {
	configMu.RLock()
	m := currentConfig.IntervalMinutes
	configMu.RUnlock()
	if m < 1 {
		return defaultInterval
	}
	return time.Duration(m) * time.Minute
}

func refreshRatesAndTooltip() error {
	refreshMu.Lock()
	defer refreshMu.Unlock()

	configMu.RLock()
	cfg := currentConfig
	configMu.RUnlock()

	if len(cfg.Pairs) == 0 {
		systray.SetTooltip("No currency pairs configured")
		return nil
	}

	quotes, err := fetchQuotes(cfg.Pairs)
	if err != nil {
		ratesMu.RLock()
		have := len(rates) > 0
		ratesMu.RUnlock()
		if have {
			systray.SetTooltip(truncateTooltip("⚠ offline – last rates:\n" + lastTooltipBody))
		} else {
			systray.SetTooltip("No rates available")
		}
		return err
	}

	newRates := map[string]float64{}
	for k, q := range quotes {
		newRates[k] = q.Rate
	}

	ratesMu.Lock()
	prev := rates
	rates = newRates
	lastQuotes = quotes
	lastSuccess = time.Now()
	ratesMu.Unlock()

	var lines []string
	for _, p := range cfg.Pairs {
		key := pairKey(p.From, p.To)
		q, ok := quotes[key]
		if !ok {
			continue
		}
		arrow := ""
		if old, ok := prev[key]; ok {
			switch {
			case q.Rate > old:
				arrow = " ▲"
			case q.Rate < old:
				arrow = " ▼"
			}
		}
		lines = append(lines, fmt.Sprintf("%s %.4f%s", key, q.Rate, arrow))
	}

	if len(lines) == 0 {
		systray.SetTooltip("No rates available")
	} else {
		lastTooltipBody = strings.Join(lines, "\n")
		systray.SetTooltip(truncateTooltip(lastTooltipBody))
	}

	checkAlarms(cfg, newRates)
	return nil
}

func truncateTooltip(s string) string {
	u := utf16.Encode([]rune(s))
	if len(u) <= maxTooltipLen {
		return s
	}
	return string(utf16.Decode(u[:maxTooltipLen-1])) + "…"
}

const alarmHysteresis = 0.001

func checkAlarms(cfg Config, latest map[string]float64) {
	for _, a := range cfg.Alarms {
		key := normalizeAlarmPair(a.Pair)
		rate, ok := latest[key]
		if !ok {
			continue
		}

		dir := strings.ToLower(strings.TrimSpace(a.Direction))
		trigKey := fmt.Sprintf("%s:%.6f:%s", key, a.Target, dir)

		var hit, rearm bool
		switch dir {
		case "above":
			hit = rate >= a.Target
			rearm = rate < a.Target*(1-alarmHysteresis)
		case "below":
			hit = rate <= a.Target
			rearm = rate > a.Target*(1+alarmHysteresis)
		default:
			continue
		}

		triggeredMu.Lock()
		armed := !alarmFired[trigKey]
		if hit && armed {
			alarmFired[trigKey] = true
			triggeredMu.Unlock()

			msg := fmt.Sprintf("%s is now %.4f (target %.4f %s)", key, rate, a.Target, dir)
			_ = beeep.Notify("FX Alarm", msg, "")
			continue
		}
		if rearm {
			delete(alarmFired, trigKey)
		}
		triggeredMu.Unlock()
	}
}

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
