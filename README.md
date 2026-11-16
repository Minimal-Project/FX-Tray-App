# FX Tray App

FX Tray App is a lightweight Windows application that displays exchange rates in the system tray, performs automatic updates, and supports custom alarms. The application runs without a visible console and provides a Windows GUI for settings.

## Features

- Display exchange rates directly in the system tray
- Automatic rate updates at fixed intervals
- Manual rate refresh via the tray menu
- Settings window for managing:
  - Currency pairs
  - Alarms (above / below)
- System notifications when alarms are triggered
- Persistent configuration via JSON file
- Windows taskbar integration (AppID, custom icons)

## Technology Stack

- Go (Golang)
- systray (Tray integration)
- Walk (Windows GUI)
- beeep (Notifications)
- rsrc (for manifest and icon embedding)
- Open Exchange Rates API (open.er-api.com)

## Project Structure
```
FX Tray/
│   main.go
│   fx.go
│   config.go
│   models.go
│   ui_settings.go
│   fxtray.manifest
│   rsrc.syso
│   go.mod
│   go.sum
│
└───assets
        icon.ico
```

## Build Instructions

### 1. Embed Manifest

If changes were made to the manifest:
```bash
go install github.com/akavel/rsrc@latest
rsrc -manifest fxtray.manifest -o rsrc.syso
```

### 2. Build without Console
```bash
go build -ldflags "-H=windowsgui -X main.version=1.1.0" -o FXTray.exe .
```

The version is shown in the tray menu and in the settings window title. Without `-X main.version=…` it shows `dev`.

When changing the version, also update `version` in `fxtray.manifest` and regenerate `rsrc.syso`:
```bash
rsrc -arch amd64 -manifest fxtray.manifest -ico assets/icon.ico -o rsrc.syso
```

### 3. Run

Simply execute FXTray.exe. The application will appear as a tray icon.

## Configuration

The application creates a `fxtray.json` file on first launch. This file contains all currency pairs and alarm rules. It is automatically loaded, saved, and edited through the UI.

## Changelog

### 1.1.0
- Exchange rates now come from the Frankfurter API (api.frankfurter.dev) instead of open.er-api.com
- New grey "FX" tray icon, also embedded as the EXE icon
- Settings window: shows an error message instead of silently falling back to Notepad
- Version shown in the tray menu and settings window title

### 1.0
- Initial release
