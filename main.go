package main

import (
	"log"
	"upswatch/internal/config"
	eventlogger "upswatch/internal/event-logger"
	"upswatch/internal/eventloop"
	httpexporter "upswatch/internal/http-exporter"
	"upswatch/internal/ups"

	"upswatch/internal/machine"
)

func main() {
	config := config.NewConfig()
	ups := ups.NewHidUps(config.VendorId, config.ProductId, config.MinVoltage, config.MaxVoltage)
	machine := machine.NewLinuxMachine()
	exporter := httpexporter.NewHttpExporter()
	logger := eventlogger.NewEventLogger(config.EventReportCooldown)

	go func() {
		if err := exporter.StartServer(config.ListenAddr); err != nil {
			log.Fatalf("HTTP listener setup failed: %s", err.Error())
		}
	}()

	eventloop.SetupWithRetry(ups, config, logger)
	eventloop.RunMonitorLoop(ups, machine, exporter, logger, config)
}
