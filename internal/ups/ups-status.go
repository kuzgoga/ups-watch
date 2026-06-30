package ups

type UpsStatus struct {
	InputVoltage   float64 // [V]
	OutputVoltage  float64 // [V]
	Load           float64 // [%]
	BatteryVoltage float64 // [V]
	BatteryCharge  float64 // [%]
	Temperature    float64 // [°C]
	IsPowerCut     bool
	IsBatteryLow   bool
	IsBuzzerOn     bool
}
