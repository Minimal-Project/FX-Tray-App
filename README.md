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
go build -ldflags "-H=windowsgui" -o FXTray.exe
```

### 3. Run

Simply execute FXTray.exe. The application will appear as a tray icon.

## Configuration

The application creates a `fxtray.json` file on first launch. This file contains all currency pairs and alarm rules. It is automatically loaded, saved, and edited through the UI.
