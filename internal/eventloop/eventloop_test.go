package eventloop_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"upswatch/internal/config"
	eventlogger "upswatch/internal/event-logger"
	"upswatch/internal/eventloop"
	httpexporter "upswatch/internal/http-exporter"
	mockmachine "upswatch/internal/mocks/machine"
	mockups "upswatch/internal/mocks/ups"
	"upswatch/internal/ups"
)

func pollUntil(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("condition was not met within timeout")
}

func getExporterState(t *testing.T, e *httpexporter.HttpExporter) (map[string]interface{}, int) {
	t.Helper()
	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	res := rec.Result()
	body := map[string]interface{}{}
	if res.StatusCode == 200 {
		_ = json.NewDecoder(res.Body).Decode(&body)
	}
	return body, res.StatusCode
}

func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	return &buf
}

func TestSetupWithRetry_RetriesConnectUntilSuccess(t *testing.T) {
	buf := captureLog(t)

	m := mockups.New()
	m.ConnectErrs = []error{errors.New("connect failed 1"), errors.New("connect failed 2")}
	m.SetStatus(ups.UpsStatus{IsBuzzerOn: false}, nil)

	cfg := config.Config{PollInterval: time.Millisecond, BuzzerInitState: false}
	logger := eventlogger.NewEventLogger(0)

	done := make(chan struct{})
	go func() {
		eventloop.SetupWithRetry(m, cfg, logger)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("SetupWithRetry did not complete in time")
	}

	if got := m.ConnectCalls; got != 3 {
		t.Errorf("expected 3 Connect attempts, got %d", got)
	}
	if got := m.ToggleCalls; got != 0 {
		t.Errorf("buzzer must not toggle when states match, got %d calls", got)
	}
	if !bytes.Contains(buf.Bytes(), []byte("UPS connected and initialized successfully")) {
		t.Error("successful connection was not logged")
	}
}

func TestSetupWithRetry_QueryStatusFailureRetriesAndDisconnects(t *testing.T) {
	m := mockups.New()
	m.QueryFunc = func(call int32) (ups.UpsStatus, error) {
		if call == 1 {
			return ups.UpsStatus{}, errors.New("query failed")
		}
		return ups.UpsStatus{IsBuzzerOn: true}, nil
	}

	cfg := config.Config{PollInterval: time.Millisecond, BuzzerInitState: false}
	logger := eventlogger.NewEventLogger(0)

	done := make(chan struct{})
	go func() {
		eventloop.SetupWithRetry(m, cfg, logger)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("SetupWithRetry did not complete in time")
	}

	if got := m.ConnectCalls; got != 2 {
		t.Errorf("expected 2 Connect calls after QueryStatus failure, got %d", got)
	}
	if got := m.DisconnectCalls; got != 1 {
		t.Errorf("expected 1 Disconnect call after QueryStatus failure, got %d", got)
	}
	if got := m.ToggleCalls; got != 1 {
		t.Errorf("expected 1 ToogleBuzzer call due to state mismatch, got %d", got)
	}
}

func TestSetupWithRetry_ToggleBuzzerErrorDoesNotBlockCompletion(t *testing.T) {
	buf := captureLog(t)

	m := mockups.New()
	m.ToggleErr = errors.New("toggle failed")
	m.SetStatus(ups.UpsStatus{IsBuzzerOn: true}, nil)

	cfg := config.Config{PollInterval: time.Millisecond, BuzzerInitState: false}
	logger := eventlogger.NewEventLogger(0)

	done := make(chan struct{})
	go func() {
		eventloop.SetupWithRetry(m, cfg, logger)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ToogleBuzzer error must not block Setup completion")
	}

	if got := m.ToggleCalls; got != 1 {
		t.Errorf("expected 1 ToogleBuzzer call, got %d", got)
	}
	if !bytes.Contains(buf.Bytes(), []byte("Failed to toggle buzzer")) {
		t.Error("buzzer toggle error was not logged")
	}
}

func TestRunMonitorLoop_PowerLostThenRestoredAfterDebounce(t *testing.T) {
	buf := captureLog(t)

	m := mockups.New()
	m.SetConnected(true)
	m.SetStatus(ups.UpsStatus{IsPowerCut: true, BatteryCharge: 100}, nil)

	cfg := config.Config{
		PollInterval:           3 * time.Millisecond,
		DebounceWindowDuration: 30 * time.Millisecond,
	}
	exporter := httpexporter.NewHttpExporter()
	logger := eventlogger.NewEventLogger(0)
	mach := mockmachine.New()

	go eventloop.RunMonitorLoop(m, mach, exporter, logger, cfg)

	pollUntil(t, time.Second, func() bool {
		body, status := getExporterState(t, exporter)
		return status == 200 && body["is_power_cut"] == true && body["time_on_battery"].(float64) > 0
	})
	if !bytes.Contains(buf.Bytes(), []byte("Power from grid lost")) {
		t.Error("power loss event was not logged")
	}

	m.SetStatus(ups.UpsStatus{IsPowerCut: false, BatteryCharge: 100}, nil)

	time.Sleep(10 * time.Millisecond)
	body, status := getExporterState(t, exporter)
	if !(status == 200 && body["time_on_battery"].(float64) > 0) {
		t.Fatal("restoration must not be accepted before debounce window elapses")
	}

	pollUntil(t, time.Second, func() bool {
		body, status := getExporterState(t, exporter)
		return status == 200 && body["time_on_battery"].(float64) == 0
	})
	if !bytes.Contains(buf.Bytes(), []byte("Power from grid restored")) {
		t.Error("power restoration event was not logged")
	}
	if mach.Calls() != 0 {
		t.Error("shutdown must not be called when critical thresholds are unset")
	}
}

func TestRunMonitorLoop_ShutdownOnCriticalBatteryCharge(t *testing.T) {
	m := mockups.New()
	m.SetConnected(true)
	m.SetStatus(ups.UpsStatus{IsPowerCut: true, BatteryCharge: 10}, nil)

	cfg := config.Config{
		PollInterval:          3 * time.Millisecond,
		CriticalBatteryCharge: 20,
	}
	exporter := httpexporter.NewHttpExporter()
	logger := eventlogger.NewEventLogger(0)
	mach := mockmachine.New()

	go eventloop.RunMonitorLoop(m, mach, exporter, logger, cfg)

	pollUntil(t, time.Second, func() bool { return mach.Calls() >= 1 })
}

func TestRunMonitorLoop_ShutdownOnCriticalPowerLossDuration(t *testing.T) {
	m := mockups.New()
	m.SetConnected(true)
	m.SetStatus(ups.UpsStatus{IsPowerCut: true, BatteryCharge: 100}, nil)

	cfg := config.Config{
		PollInterval:              3 * time.Millisecond,
		CriticalPowerLossDuration: 15 * time.Millisecond,
	}
	exporter := httpexporter.NewHttpExporter()
	logger := eventlogger.NewEventLogger(0)
	mach := mockmachine.New()

	go eventloop.RunMonitorLoop(m, mach, exporter, logger, cfg)

	pollUntil(t, time.Second, func() bool { return mach.Calls() >= 1 })
}

func TestRunMonitorLoop_ReconnectsAfterConnectionLost(t *testing.T) {
	buf := captureLog(t)

	m := mockups.New()
	m.ConnectErrs = []error{errors.New("connect failed")}
	m.SetStatus(ups.UpsStatus{IsPowerCut: false}, nil)

	cfg := config.Config{PollInterval: 3 * time.Millisecond}
	exporter := httpexporter.NewHttpExporter()
	logger := eventlogger.NewEventLogger(0)
	mach := mockmachine.New()

	go eventloop.RunMonitorLoop(m, mach, exporter, logger, cfg)

	pollUntil(t, time.Second, func() bool {
		_, status := getExporterState(t, exporter)
		return status == 200
	})

	if !bytes.Contains(buf.Bytes(), []byte("Connection to UPS lost/failed")) {
		t.Error("initial failed connection attempt was not logged")
	}
	if !bytes.Contains(buf.Bytes(), []byte("Connection to UPS restored")) {
		t.Error("connection restoration was not logged")
	}
}

func TestRunMonitorLoop_QueryFailureDisconnectsAndResetsExporter(t *testing.T) {
	m := mockups.New()
	m.SetConnected(true)
	m.QueryFunc = func(call int32) (ups.UpsStatus, error) {
		return ups.UpsStatus{}, errors.New("query always fails")
	}

	cfg := config.Config{PollInterval: 3 * time.Millisecond}
	exporter := httpexporter.NewHttpExporter()
	logger := eventlogger.NewEventLogger(0)
	mach := mockmachine.New()

	go eventloop.RunMonitorLoop(m, mach, exporter, logger, cfg)

	pollUntil(t, time.Second, func() bool {
		return m.DisconnectCalls >= 1
	})

	_, status := getExporterState(t, exporter)
	if status != 503 {
		t.Errorf("exporter must report UPS unavailable after query failure, got status %d", status)
	}
}
