package httpexporter

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
	"upswatch/internal/ups"
)

type HttpExporter struct {
	mu            sync.RWMutex
	status        ups.UpsStatus
	isConnected   bool
	timeOnBattery time.Duration
}

func NewHttpExporter() *HttpExporter {
	return &HttpExporter{}
}

func (e *HttpExporter) UpdateState(status ups.UpsStatus, isConnected bool, timeOnBattery time.Duration) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.status = status
	e.isConnected = isConnected
	e.timeOnBattery = timeOnBattery
}

func (e *HttpExporter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.isConnected {
		http.Error(w, "UPS is unavailable", http.StatusServiceUnavailable)
		return
	}

	metrics := map[string]any{
		"input_voltage":   e.status.InputVoltage,
		"output_voltage":  e.status.OutputVoltage,
		"load":            e.status.Load,
		"battery_voltage": e.status.BatteryVoltage,
		"battery_charge":  e.status.BatteryCharge,
		"temperature":     e.status.Temperature,
		"is_power_cut":    e.status.IsPowerCut,
		"is_battery_low":  e.status.IsBatteryLow,
		"is_buzzer_on":    e.status.IsBuzzerOn,
		"time_on_battery": e.timeOnBattery.Seconds(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(metrics)
}

func (e *HttpExporter) StartServer(addr string) error {
	http.Handle("/metrics", e)
	return http.ListenAndServe(addr, nil)
}
