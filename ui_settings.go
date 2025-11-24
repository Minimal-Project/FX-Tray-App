package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

// Währungen für Vorschläge
var currencySuggestions = []string{
	"CHF", "EUR", "USD", "GBP", "JPY", "CAD", "AUD", "NZD",
	"SEK", "NOK", "DKK", "PLN", "CZK", "HUF", "TRY", "CNY",
	"INR", "BRL", "MXN", "ZAR", "KRW", "SGD", "HKD", "THB",
}

// Öffnet Settings-Fenster
func openSettingsWindow() {
	configMu.RLock()
	cfg := currentConfig
	configMu.RUnlock()

	pairModel := NewPairTableModel(cfg.Pairs)
	alarmModel := NewAlarmTableModel(cfg.Alarms)

	var mainWindow *walk.MainWindow
	var pairTable *walk.TableView
	var alarmTable *walk.TableView
	var statusLabel *walk.Label

	// Pair hinzufügen Dialog
	addPairFunc := func() {
		var dlg *walk.Dialog
		var fromCombo, toCombo *walk.ComboBox
		var selectedFrom, selectedTo string

		result, err := Dialog{
			AssignTo: &dlg,
			Title:    "Add Currency Pair",
			MinSize:  Size{Width: 300, Height: 150},
			Layout:   VBox{},
			Children: []Widget{
				Composite{
					Layout: Grid{Columns: 2},
					Children: []Widget{
						Label{Text: "From:"},
						ComboBox{
							AssignTo: &fromCombo,
							Editable: true,
							Model:    currencySuggestions,
						},
						Label{Text: "To:"},
						ComboBox{
							AssignTo: &toCombo,
							Editable: true,
							Model:    currencySuggestions,
						},
					},
				},
				Composite{
					Layout: HBox{},
					Children: []Widget{
						HSpacer{},
						PushButton{
							Text: "Add",
							OnClicked: func() {
								selectedFrom = strings.TrimSpace(fromCombo.Text())
								selectedTo = strings.TrimSpace(toCombo.Text())

								if selectedFrom == "" || selectedTo == "" {
									walk.MsgBox(dlg, "Validation",
										"Please enter both currencies.",
										walk.MsgBoxIconWarning)
									return
								}

								dlg.Accept()
							},
						},
						PushButton{
							Text:      "Cancel",
							OnClicked: func() { dlg.Cancel() },
						},
					},
				},
			},
		}.Run(mainWindow)

		if err != nil {
			walk.MsgBox(mainWindow, "Error", "Failed to open dialog: "+err.Error(), walk.MsgBoxIconError)
			return
		}

		if result == walk.DlgCmdOK && selectedFrom != "" && selectedTo != "" {
			pairModel.items = append(pairModel.items, PairRow{
				From: strings.ToUpper(selectedFrom),
				To:   strings.ToUpper(selectedTo),
			})
			pairModel.PublishRowsReset()
			pairTable.SetCurrentIndex(len(pairModel.items) - 1)
		}
	}

	// Pair löschen
	deletePairFunc := func() {
		idx := pairTable.CurrentIndex()
		if idx >= 0 && idx < len(pairModel.items) {
			pairModel.items = append(pairModel.items[:idx], pairModel.items[idx+1:]...)
			pairModel.PublishRowsReset()
		}
	}

	// Alarm hinzufügen Dialog
	addAlarmFunc := func() {
		var dlg *walk.Dialog
		var pairEdit *walk.LineEdit
		var targetEdit *walk.NumberEdit
		var dirCombo *walk.ComboBox

		directions := []string{"above", "below"}

		defaultPair := ""
		if pairTable != nil {
			if idx := pairTable.CurrentIndex(); idx >= 0 && idx < len(pairModel.items) {
				pr := pairModel.items[idx]
				defaultPair = strings.ToUpper(pr.From) + "/" + strings.ToUpper(pr.To)
			}
		}

		var selectedPair string
		var selectedTarget float64
		var selectedDirection string

		result, err := Dialog{
			AssignTo: &dlg,
			Title:    "Add Alarm",
			MinSize:  Size{Width: 300, Height: 180},
			Layout:   VBox{},
			Children: []Widget{
				Composite{
					Layout: Grid{Columns: 2},
					Children: []Widget{
						Label{Text: "Pair (e.g. EUR/CHF):"},
						LineEdit{
							AssignTo: &pairEdit,
							Text:     defaultPair,
						},
						Label{Text: "Target:"},
						NumberEdit{
							AssignTo: &targetEdit,
							Decimals: 4,
						},
						Label{Text: "Direction:"},
						ComboBox{
							AssignTo:     &dirCombo,
							Model:        directions,
							CurrentIndex: 0,
						},
					},
				},
				Composite{
					Layout: HBox{},
					Children: []Widget{
						HSpacer{},
						PushButton{
							Text: "Add",
							OnClicked: func() {
								selectedPair = strings.TrimSpace(pairEdit.Text())
								selectedTarget = targetEdit.Value()

								idx := dirCombo.CurrentIndex()
								if idx >= 0 && idx < len(directions) {
									selectedDirection = directions[idx]
								}

								if selectedPair == "" {
									walk.MsgBox(dlg, "Validation",
										"Please enter a currency pair (e.g. EUR/CHF).",
										walk.MsgBoxIconWarning)
									return
								}
								if selectedTarget == 0 {
									walk.MsgBox(dlg, "Validation",
										"Please enter a non-zero target.",
										walk.MsgBoxIconWarning)
									return
								}
								if selectedDirection == "" {
									walk.MsgBox(dlg, "Validation",
										"Please select a direction (above/below).",
										walk.MsgBoxIconWarning)
									return
								}

								dlg.Accept()
							},
						},
						PushButton{
							Text: "Cancel",
							OnClicked: func() {
								dlg.Cancel()
							},
						},
					},
				},
			},
		}.Run(mainWindow)

		if err != nil {
			walk.MsgBox(mainWindow, "Error", "Failed to open dialog: "+err.Error(), walk.MsgBoxIconError)
			return
		}

		if result == walk.DlgCmdOK && selectedPair != "" && selectedDirection != "" {
			alarmModel.items = append(alarmModel.items, AlarmRow{
				Pair:      normalizeAlarmPair(selectedPair),
				Target:    selectedTarget,
				Direction: selectedDirection,
			})
			alarmModel.PublishRowsReset()
			alarmTable.SetCurrentIndex(len(alarmModel.items) - 1)

		}
	}

	// Alarm bearbeiten
	editAlarmFunc := func() {
		idx := alarmTable.CurrentIndex()
		if idx < 0 || idx >= len(alarmModel.items) {
			walk.MsgBox(mainWindow, "Info", "Please select an alarm to edit.", walk.MsgBoxIconInformation)
			return
		}

		currentAlarm := alarmModel.items[idx]

		var dlg *walk.Dialog
		var pairEdit *walk.LineEdit
		var targetEdit *walk.NumberEdit
		var dirCombo *walk.ComboBox

		directions := []string{"above", "below"}
		currentDirIndex := 0
		if strings.ToLower(currentAlarm.Direction) == "below" {
			currentDirIndex = 1
		}

		var selectedPair string
		var selectedTarget float64
		var selectedDirection string

		result, err := Dialog{
			AssignTo: &dlg,
			Title:    "Edit Alarm",
			MinSize:  Size{Width: 300, Height: 180},
			Layout:   VBox{},
			Children: []Widget{
				Composite{
					Layout: Grid{Columns: 2},
					Children: []Widget{
						Label{Text: "Pair (e.g. EUR/CHF):"},
						LineEdit{
							AssignTo: &pairEdit,
							Text:     currentAlarm.Pair,
						},
						Label{Text: "Target:"},
						NumberEdit{
							AssignTo: &targetEdit,
							Value:    currentAlarm.Target,
							Decimals: 4,
						},
						Label{Text: "Direction:"},
						ComboBox{
							AssignTo:     &dirCombo,
							Model:        directions,
							CurrentIndex: currentDirIndex,
						},
					},
				},
				Composite{
					Layout: HBox{},
					Children: []Widget{
						HSpacer{},
						PushButton{
							Text: "Save",
							OnClicked: func() {
								selectedPair = strings.TrimSpace(pairEdit.Text())
								selectedTarget = targetEdit.Value()

								dirIdx := dirCombo.CurrentIndex()
								if dirIdx >= 0 && dirIdx < len(directions) {
									selectedDirection = directions[dirIdx]
								}

								if selectedPair == "" {
									walk.MsgBox(dlg, "Validation",
										"Please enter a currency pair (e.g. EUR/CHF).",
										walk.MsgBoxIconWarning)
									return
								}
								if selectedTarget == 0 {
									walk.MsgBox(dlg, "Validation",
										"Please enter a non-zero target.",
										walk.MsgBoxIconWarning)
									return
								}
								if selectedDirection == "" {
									walk.MsgBox(dlg, "Validation",
										"Please select a direction (above/below).",
										walk.MsgBoxIconWarning)
									return
								}

								dlg.Accept()
							},
						},
						PushButton{
							Text: "Cancel",
							OnClicked: func() {
								dlg.Cancel()
							},
						},
					},
				},
			},
		}.Run(mainWindow)

		if err != nil {
			walk.MsgBox(mainWindow, "Error", "Failed to open dialog: "+err.Error(), walk.MsgBoxIconError)
			return
		}

		if result == walk.DlgCmdOK && selectedPair != "" && selectedDirection != "" {
			alarmModel.items[idx] = AlarmRow{
				Pair:      normalizeAlarmPair(selectedPair),
				Target:    selectedTarget,
				Direction: selectedDirection,
			}
			alarmModel.PublishRowsReset()
			alarmTable.SetCurrentIndex(idx)

		}
	}

	// Alarm löschen
	deleteAlarmFunc := func() {
		idx := alarmTable.CurrentIndex()
		if idx >= 0 && idx < len(alarmModel.items) {
			alarmModel.items = append(alarmModel.items[:idx], alarmModel.items[idx+1:]...)
			alarmModel.PublishRowsReset()
		}
	}

	// Speichern
	saveFunc := func() {
		var newCfg Config

		for _, p := range pairModel.items {
			newCfg.Pairs = append(newCfg.Pairs, CurrencyPair{
				From: p.From,
				To:   p.To,
			})
		}

		for _, a := range alarmModel.items {
			newCfg.Alarms = append(newCfg.Alarms, Alarm{
				Pair:      a.Pair,
				Target:    a.Target,
				Direction: a.Direction,
			})
		}

		if err := saveConfig(newCfg); err != nil {
			walk.MsgBox(mainWindow, "Error", "Failed to save config: "+err.Error(), walk.MsgBoxIconError)
			return
		}

		statusLabel.SetText("Saved successfully!")
		go func() {
			time.Sleep(2 * time.Second)
			statusLabel.SetText("")
		}()

		// Nach Speichern Kurse neu laden
		go func() {
			if err := refreshRatesAndTooltip(); err != nil {
				fmt.Println("refresh after save:", err)
			}
		}()
	}

	// Hauptfenster
	_, err := MainWindow{
		AssignTo: &mainWindow,
		Title:    "FX Tray Settings",

		Size:    Size{Width: 290, Height: 380},
		MinSize: Size{Width: 250, Height: 300},

		Layout: VBox{Margins: Margins{Left: 6, Top: 6, Right: 6, Bottom: 6}},
		Children: []Widget{
			Label{
				Text: "Currency Pairs",
				Font: Font{PointSize: 10, Bold: true},
			},
			TableView{
				AssignTo:         &pairTable,
				AlternatingRowBG: true,
				MinSize:          Size{Width: 0, Height: 100},
				Columns: []TableViewColumn{
					{Title: "From", Width: 80},
					{Title: "To", Width: 80},
				},
				Model: pairModel,
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					PushButton{
						Text:      "Add Pair",
						OnClicked: addPairFunc,
					},
					PushButton{
						Text:      "Delete Selected",
						OnClicked: deletePairFunc,
					},
					HSpacer{},
				},
			},
			VSpacer{Size: 8},
			Label{
				Text: "Alarms",
				Font: Font{PointSize: 10, Bold: true},
			},
			TableView{
				AssignTo:         &alarmTable,
				AlternatingRowBG: true,
				MinSize:          Size{Width: 0, Height: 100},
				Columns: []TableViewColumn{
					{Title: "Pair", Width: 100},
					{Title: "Target", Width: 80},
					{Title: "Direction", Width: 80},
				},
				Model: alarmModel,
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					PushButton{
						Text:      "Add Alarm",
						OnClicked: addAlarmFunc,
					},
					PushButton{
						Text:      "Edit Selected",
						OnClicked: editAlarmFunc,
					},
					PushButton{
						Text:      "Delete Selected",
						OnClicked: deleteAlarmFunc,
					},
					HSpacer{},
				},
			},
			VSpacer{Size: 8},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					PushButton{
						Text:      "Save",
						OnClicked: saveFunc,
					},
					Label{
						AssignTo: &statusLabel,
						Text:     "",
					},
					HSpacer{},
				},
			},
		},
	}.Run()

	if err != nil {

		// Fallback
		url := "file://" + configPath
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "windows":
			cmd = exec.Command("notepad", configPath)
		case "darwin":
			cmd = exec.Command("open", url)
		default:
			cmd = exec.Command("xdg-open", url)
		}
		_ = cmd.Start()
		return
	}
}
