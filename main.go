package main

import (
	"log"
	"upswatch/internal/config"
	"upswatch/internal/ups"

	"github.com/k0kubun/pp/v3"
)

func SetupBuzzer(ups ups.UpsInterface, buzzerInitState bool) error {
	status, err := ups.QueryStatus()
	if err != nil {
		return err
	}

	if status.IsBuzzerOn != buzzerInitState {
		if err = ups.ToogleBuzzer(); err != nil {
			return err
		}
	}
	return nil
}

func Setup(ups ups.UpsInterface, config config.Config) {
	err := ups.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to ups: %s", err.Error())
	}

	if err := SetupBuzzer(ups, config.BuzzerInitState); err != nil {
		log.Fatalf("Failed to init buzzer: %s", err.Error())
	}
}

func main() {
	config := config.NewConfig()
	ups := ups.NewHidUps(config.VendorId, config.ProductId, config.MinVoltage, config.MaxVoltage)

	Setup(ups, config)

	status, _ := ups.QueryStatus()
	pp.Print(status)
}
