package main

import (
	"fmt"

	"github.com/lxn/walk"
)

// Definitionen
// PairRow für UI-Tabelle
type PairRow struct {
	From string
	To   string
}

// AlarmRow für UI-Tabelle
type AlarmRow struct {
	Pair      string
	Target    float64
	Direction string
}

// TableModel für Currency Pairs
type PairTableModel struct {
	walk.TableModelBase
	items []PairRow
}

// TableModel aus Config
func NewPairTableModel(pairs []CurrencyPair) *PairTableModel {
	m := &PairTableModel{}
	for _, p := range pairs {
		m.items = append(m.items, PairRow{From: p.From, To: p.To})
	}
	return m
}

func (m *PairTableModel) RowCount() int {
	return len(m.items)
}

func (m *PairTableModel) Value(row, col int) interface{} {
	item := m.items[row]
	switch col {
	case 0:
		return item.From
	case 1:
		return item.To
	}
	return ""
}

// TableModel für Alarme
type AlarmTableModel struct {
	walk.TableModelBase
	items []AlarmRow
}

// TableModel aus Config
func NewAlarmTableModel(alarms []Alarm) *AlarmTableModel {
	m := &AlarmTableModel{}
	for _, a := range alarms {
		m.items = append(m.items, AlarmRow{
			Pair:      a.Pair,
			Target:    a.Target,
			Direction: a.Direction,
		})
	}
	return m
}

func (m *AlarmTableModel) RowCount() int {
	return len(m.items)
}

func (m *AlarmTableModel) Value(row, col int) interface{} {
	item := m.items[row]
	switch col {
	case 0:
		return item.Pair
	case 1:
		return fmt.Sprintf("%.4f", item.Target)
	case 2:
		return item.Direction
	}
	return ""
}
