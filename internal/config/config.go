package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	VendorId        uint16
	ProductId       uint16
	MinVoltage      float64
	MaxVoltage      float64
	BuzzerInitState bool
}

func getRequiredEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatalf("%s should be specified", name)
	}
	return value
}

func getOptionalEnv(name, defaultValue string) string {
	value := os.Getenv(name)
	if value == "" {
		return defaultValue
	}
	return value
}

func NewConfig() Config {
	godotenv.Load()

	vendorId, _ := strconv.ParseUint(getRequiredEnv("VENDOR_ID"), 16, 16)
	productId, _ := strconv.ParseUint(getRequiredEnv("PRODUCT_ID"), 16, 16)
	minVoltage, _ := strconv.ParseFloat(getRequiredEnv("MIN_VOLTAGE"), 64)
	maxVoltage, _ := strconv.ParseFloat(getRequiredEnv("MAX_VOLTAGE"), 64)
	buzzerInitState := strings.ToLower(getOptionalEnv("BUZZER_INIT_STATE", "false")) == "true"

	return Config{
		VendorId:        uint16(vendorId),
		ProductId:       uint16(productId),
		MinVoltage:      minVoltage,
		MaxVoltage:      maxVoltage,
		BuzzerInitState: buzzerInitState,
	}
}
