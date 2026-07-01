package ups

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/sstallion/go-hid"
)

type HidUps struct {
	vendorId   uint16
	productId  uint16
	device     *hid.Device
	minVoltage float64
	maxVoltage float64
}

func NewHidUps(vendorId, productId uint16, minVoltage, maxVoltage float64) *HidUps {
	return &HidUps{
		vendorId,
		productId,
		nil,
		minVoltage,
		maxVoltage,
	}
}

func (ups *HidUps) Connect() error {
	device, err := hid.OpenFirst(ups.vendorId, ups.productId)
	if err != nil {
		return fmt.Errorf("failed to open device(VendorID: 0x%04x, ProductID: 0x%04x): %w", ups.vendorId, ups.productId, err)
	}
	ups.device = device
	return nil
}

func (ups *HidUps) Disconnect() error {
	if ups.device != nil {
		err := ups.device.Close()
		ups.device = nil
		return err
	}
	return nil
}

func (ups *HidUps) IsConnected() bool {
	return ups.device != nil
}

func (u *HidUps) sendRawQuery(cmdStr string) (string, error) {
	if u.device == nil {
		return "", fmt.Errorf("device is closed")
	}

	command := make([]byte, 9)
	command[0] = 0x00
	copy(command[1:], []byte(cmdStr))

	if _, err := u.device.Write(command); err != nil {
		return "", err
	}

	var fullResponse []byte
	buf := make([]byte, 64)

	for range 10 {
		n, err := u.device.Read(buf)
		if err != nil {
			return "", err
		}
		if n > 0 {
			for _, b := range buf[:n] {
				if b != 0 {
					fullResponse = append(fullResponse, b)
				}
			}
			if bytes.Contains(buf[:n], []byte{'\r'}) {
				str := string(fullResponse)
				idx := strings.LastIndex(str, "(")
				if idx != -1 {
					return strings.TrimSpace(str[idx:]), nil
				}
			}
		}
	}
	return "", fmt.Errorf("timeout error")
}

func (ups *HidUps) QueryStatus() (UpsStatus, error) {
	res, err := ups.sendRawQuery("Q1\r")
	if err == nil {
		return ups.parseQ1(res)
	}
	return UpsStatus{}, err
}

func (ups *HidUps) parseQ1(response string) (UpsStatus, error) {
	parts := strings.Fields(strings.Trim(response, "()"))
	if len(parts) < 8 {
		return UpsStatus{}, fmt.Errorf("invalid response: %q", response)
	}
	inputVoltage, _ := strconv.ParseFloat(parts[0], 64)
	outputVoltage, _ := strconv.ParseFloat(parts[2], 64)
	load, _ := strconv.ParseFloat(parts[3], 64)
	batteryVoltage, _ := strconv.ParseFloat(parts[5], 64)
	temperature, _ := strconv.ParseFloat(parts[6], 64)
	bits := parts[7]

	batteryCharge := (batteryVoltage - ups.minVoltage) / (ups.maxVoltage - ups.minVoltage) * 100
	batteryCharge = max(min(batteryCharge, 100), 0)

	return UpsStatus{
		InputVoltage:   inputVoltage,
		OutputVoltage:  outputVoltage,
		Load:           load,
		BatteryVoltage: batteryVoltage,
		BatteryCharge:  batteryCharge,
		Temperature:    temperature,
		IsPowerCut:     bits[0] == '1',
		IsBatteryLow:   bits[1] == '1',
		IsBuzzerOn:     bits[7] == '1',
	}, nil
}

func (ups *HidUps) ToogleBuzzer() error {
	if !ups.IsConnected() {
		return fmt.Errorf("device is closed")
	}

	command := make([]byte, 8)
	copy(command[1:], []byte("Q\r"))
	_, err := ups.device.Write(command)
	return err
}
