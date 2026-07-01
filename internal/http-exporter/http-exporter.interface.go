package httpexporter

import (
	"time"
	"upswatch/internal/ups"
)

type HttpExporterInterface interface {
	UpdateState(status ups.UpsStatus, isConnected bool, powerCutDuration time.Duration)
	StartServer(addr string) error
}
