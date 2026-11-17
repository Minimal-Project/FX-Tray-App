# FX Tray App

FX Tray App is a lightweight Windows application that displays exchange rates in the system tray, performs automatic updates, and supports custom alarms.

## Features

- Display exchange rates directly in the system tray
- Intraday market rates from free sources, no API key required
- Automatic rate updates at a configurable interval (default 5 minutes)
- Trend arrows (▲ ▼) compared to the previous update
- Manual rate refresh via the tray menu
- "Last Updated" with time of the last successful fetch and its source
- Last known rates stay visible (marked offline) when no source is reachable
- Settings window for managing:
  - Currency pairs
  - Alarms (above / below)
- System notifications when an alarm target is crossed
- Persistent configuration via JSON file
- Windows taskbar integration

## Rate Sources

Rates are fetched through a provider chain. If a source fails or doesn't know a pair, the next one is tried:

| Order | Source | Update frequency | Notes |
|---|---|---|---|
| 1 | Swissquote public quote feed | Live (market hours) | Mid rate of the tightest bid/ask spread |
| 2 | Coinbase exchange rates API | Continuously | Many fiat currencies |
| 3 | Frankfurter API (api.frankfurter.dev) | Daily | Official reference rates, fallback |

All sources are free and require no API key. The Swissquote feed is public but not an official API and may change without notice.

## Technology Stack

- Go (Golang)
- systray (Tray integration)
- Walk (Windows GUI)
- beeep (Notifications)
- rsrc (for manifest and icon embedding)

## Project Structure
```
FX Tray/
│   main.go          Tray setup, menu, state
│   fx.go            Update loop, tooltip, alarms
│   providers.go     Rate sources (Swissquote, Coinbase, Frankfurter)
│   config.go        Config file handling
│   models.go        Table models for the settings UI
│   ui_settings.go   Settings window
│   fxtray.manifest
│   rsrc.syso
│   go.mod
│   go.sum
│
└───assets
        icon.ico
```

## Build Instructions

### 1. Embed Manifest and Icon

Install rsrc:
```powershell
go install github.com/akavel/rsrc@latest
```

Go installs tools to `%USERPROFILE%\go\bin`. If `rsrc` is not found, add that folder to your PATH:
```powershell
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";$env:USERPROFILE\go\bin", "User")
```

When changing the version, update `version` in `fxtray.manifest` and regenerate `rsrc.syso`:
```powershell
rsrc -arch amd64 -manifest fxtray.manifest -ico assets/icon.ico -o rsrc.syso
```

### 2. Build
```powershell
go build -ldflags "-s -w -H=windowsgui -X main.version=1.2.0" -o FXTray.exe .
```

### 3. Run

Simply execute FXTray.exe. The application will appear as a tray icon.

## Configuration

The application creates a `fxtray.json` file next to the EXE on first launch. Currency pairs and alarms are edited through the settings window.

Example:
```json
{
  "pairs": [
    { "from": "CHF", "to": "EUR" },
    { "from": "EUR", "to": "CHF" }
  ],
  "alarms": [
    { "pair": "EUR/CHF", "target": 0.95, "direction": "above" }
  ],
  "intervalMinutes": 5
}
```

| Field | Description |
|---|---|
| `pairs` | Currency pairs shown in the tooltip |
| `alarms` | `direction` is `above` or `below`. An alarm fires once when the target is crossed and re-arms after the rate moves back by 0.1 % |
| `intervalMinutes` | Optional. Auto-update interval in minutes (default 5, minimum 1). Not editable in the UI yet |

Note: Windows limits tray tooltips to 127 characters. With many pairs, the tooltip is trimmed.

## Changelog

### 1.2.0
- Intraday rates via provider chain: Swissquote → Coinbase → Frankfurter (all free, no API key)
- Configurable update interval (`intervalMinutes`)
- "Last Updated" shows the time of the last successful fetch and the source
- Manual refresh restarts the auto-update timer
- Alarms fire once when the target is crossed (re-armed after 0.1 % pullback) instead of every 5 minutes
- Trend arrows ▲▼ in the tooltip; tooltip trimmed to the Windows limit of 127 characters
- Last rates stay visible (marked offline) when all sources fail
- A failing request for one base currency no longer discards all rates

### 1.1.0
- Exchange rates now come from the Frankfurter API (api.frankfurter.dev) instead of open.er-api.com
- New grey "FX" tray icon, also embedded as the EXE icon
- Settings window: shows an error message instead of silently falling back to Notepad
- Version shown in the tray menu and settings window title

### 1.0
- Initial release
