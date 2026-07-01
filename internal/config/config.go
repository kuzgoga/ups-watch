package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	VendorId                  uint16
	ProductId                 uint16
	MinVoltage                float64
	MaxVoltage                float64
	BuzzerInitState           bool
	PollInterval              time.Duration
	CriticalBatteryCharge     int
	CriticalPowerLossDuration time.Duration
	ListenAddr                string
	EventReportCooldown       time.Duration
	DebounceWindowDuration    time.Duration
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

func getOptionalDurationEnv(name, defaultValue string) time.Duration {
	value := os.Getenv(name)
	if value == "" {
		value = defaultValue
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		log.Fatalf("Failed to read %s value as duration", name)
	}
	return duration
}

func NewConfig() Config {
	_ = godotenv.Load()

	vendorId, _ := strconv.ParseUint(getRequiredEnv("VENDOR_ID"), 16, 16)
	productId, _ := strconv.ParseUint(getRequiredEnv("PRODUCT_ID"), 16, 16)
	minVoltage, _ := strconv.ParseFloat(getRequiredEnv("MIN_VOLTAGE"), 64)
	maxVoltage, _ := strconv.ParseFloat(getRequiredEnv("MAX_VOLTAGE"), 64)
	buzzerInitState := strings.ToLower(getOptionalEnv("BUZZER_INIT_STATE", "false")) == "true"
	criticalBatteryCharge, _ := strconv.ParseInt(getOptionalEnv("CRITICAL_BATTERY_CHARGE", "0"), 10, 64)
	criticalPowerLossDuration, _ := time.ParseDuration(getOptionalEnv("CRITICAL_POWER_LOSS_DURATION", "0s"))
	listenAddr := getOptionalEnv("LISTEN_ADDRESS", "127.0.0.1:8080")
	pollInterval := getOptionalDurationEnv("POLL_INTERVAL", "5s")
	eventReportCooldown := getOptionalDurationEnv("EVENT_REPORT_COOLDOWN", "10s")
	debounceWindowDuration := getOptionalDurationEnv("DEBOUNCE_WINDOW_DURATION", "10s")

	return Config{
		VendorId:                  uint16(vendorId),
		ProductId:                 uint16(productId),
		MinVoltage:                minVoltage,
		MaxVoltage:                maxVoltage,
		BuzzerInitState:           buzzerInitState,
		PollInterval:              pollInterval,
		CriticalBatteryCharge:     int(criticalBatteryCharge),
		CriticalPowerLossDuration: criticalPowerLossDuration,
		ListenAddr:                listenAddr,
		EventReportCooldown:       eventReportCooldown,
		DebounceWindowDuration:    debounceWindowDuration,
	}
}
