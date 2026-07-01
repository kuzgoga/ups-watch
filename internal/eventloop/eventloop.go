package eventloop

import (
	"log"
	"time"
	"upswatch/internal/config"
	eventlogger "upswatch/internal/event-logger"
	httpexporter "upswatch/internal/http-exporter"
	"upswatch/internal/machine"
	"upswatch/internal/ups"
)

func SetupWithRetry(upsClient ups.UpsInterface, config config.Config, logger *eventlogger.EventLogger) {
	for {
		err := upsClient.Connect()
		if err != nil {
			logger.Log(eventlogger.EventUpsConnectionLost, "Setup: Failed to connect to UPS: "+err.Error())
			time.Sleep(config.PollInterval)
			continue
		}

		status, err := upsClient.QueryStatus()
		if err != nil {
			logger.Log(eventlogger.EventUpsConnectionLost, "Setup: Failed to query UPS status: "+err.Error())
			upsClient.Disconnect()
			time.Sleep(config.PollInterval)
			continue
		}

		if status.IsBuzzerOn != config.BuzzerInitState {
			if err = upsClient.ToogleBuzzer(); err != nil {
				log.Printf("Setup: Failed to toggle buzzer: %s\n", err.Error())
			}
		}

		logger.Log(eventlogger.EventUpsConnected, "Setup: UPS connected and initialized successfully")
		break
	}
}

func RunMonitorLoop(
	upsClient ups.UpsInterface,
	machine machine.MachineInterface,
	exporter *httpexporter.HttpExporter,
	logger *eventlogger.EventLogger,
	config config.Config,
) {
	var (
		isPowerCutReal bool
		powerLostStart time.Time
		debounceStart  time.Time
	)

	for {
		if !upsClient.IsConnected() {
			exporter.UpdateState(ups.UpsStatus{}, false, 0)
			if err := upsClient.Connect(); err != nil {
				logger.Log(eventlogger.EventUpsConnectionLost, "Connection to UPS lost/failed: "+err.Error())
				time.Sleep(config.PollInterval)
				continue
			}
			logger.Log(eventlogger.EventUpsConnected, "Connection to UPS restored")
		}

		status, err := upsClient.QueryStatus()
		if err != nil {
			logger.Log(eventlogger.EventUpsConnectionLost, "Failed to query UPS: "+err.Error())
			upsClient.Disconnect()
			exporter.UpdateState(ups.UpsStatus{}, false, 0)
			time.Sleep(config.PollInterval)
			continue
		}

		now := time.Now()

		if status.IsPowerCut {
			debounceStart = time.Time{}

			if !isPowerCutReal {
				isPowerCutReal = true
				powerLostStart = now
				logger.Log(eventlogger.EventPowerLost, "Power from grid lost. Operating on battery.")
			}
		} else {
			if isPowerCutReal {
				if debounceStart.IsZero() {
					debounceStart = now
				} else if now.Sub(debounceStart) >= config.DebounceWindowDuration {
					isPowerCutReal = false
					powerLostStart = time.Time{}
					logger.Log(eventlogger.EventPowerRestored, "Power from grid restored securely.")
				}
			}
		}

		var powerLossDuration time.Duration
		if isPowerCutReal {
			powerLossDuration = now.Sub(powerLostStart)
		}

		exporter.UpdateState(status, true, powerLossDuration)

		if isPowerCutReal {
			minsOnBat := powerLossDuration.Minutes()
			isCriticalBatteryCharge := config.CriticalBatteryCharge != 0 && status.BatteryCharge < float64(config.CriticalBatteryCharge)
			isCriticalPowerLossDuration := config.CriticalPowerLossDuration.Seconds() != 0 && minsOnBat > config.CriticalPowerLossDuration.Minutes()

			if isCriticalBatteryCharge || isCriticalPowerLossDuration {
				logger.Log(eventlogger.EventShutdown, "Shutdown machine")
				machine.Shutdown()
			}
		}

		time.Sleep(config.PollInterval)
	}
}
