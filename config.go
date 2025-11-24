package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

//Definitionen

// Währungspaar definition
type CurrencyPair struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Alarm definition
type Alarm struct {
	Pair      string  `json:"pair"`
	Target    float64 `json:"target"`
	Direction string  `json:"direction"`
}

// Config definition
type Config struct {
	Pairs  []CurrencyPair `json:"pairs"`
	Alarms []Alarm        `json:"alarms"`
}

// Antwort API https://open.er-api.com/v6/latest/{BASE}
type rateResponse struct {
	Result   string             `json:"result"`
	BaseCode string             `json:"base_code"`
	Rates    map[string]float64 `json:"rates"`
}

// Config-Datei

// Config Pfad
func defaultConfigPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "fxtray.json"
	}
	dir := filepath.Dir(exe)
	return filepath.Join(dir, "fxtray.json")
}

// Config anlegen
func ensureConfig() error {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		def := Config{
			Pairs: []CurrencyPair{
				{From: "CHF", To: "EUR"},
				{From: "EUR", To: "CHF"},
			},
			Alarms: []Alarm{},
		}
		data, err := json.MarshalIndent(def, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(configPath, data, 0644)
	}
	return nil
}

// Config laden
func loadConfig() error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}
	configMu.Lock()
	currentConfig = cfg
	configMu.Unlock()
	return nil
}

// Config aktualisieren
func saveConfig(cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return err
	}
	configMu.Lock()
	currentConfig = cfg
	configMu.Unlock()
	return nil
}
