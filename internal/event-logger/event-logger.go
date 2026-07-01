package eventlogger

import (
	"log"
	"sync"
	"time"
)

type EventType string

const (
	EventUpsConnectionLost EventType = "UpsConnectionLost"
	EventUpsConnected      EventType = "UpsConnected"
	EventPowerLost         EventType = "PowerLost"
	EventPowerRestored     EventType = "PowerRestored"
	EventShutdown          EventType = "Shutdown"
)

type EventLogger struct {
	lastEventTimes   map[EventType]time.Time
	mu               sync.Mutex
	eventLogCooldown time.Duration
}

func NewEventLogger(eventLogCooldown time.Duration) *EventLogger {
	return &EventLogger{
		lastEventTimes:   make(map[EventType]time.Time),
		eventLogCooldown: eventLogCooldown,
	}
}

func (el *EventLogger) Notify() {
	// TODO:
}

func (el *EventLogger) Log(eventType EventType, msg string) {
	el.mu.Lock()
	defer el.mu.Unlock()

	now := time.Now()
	if now.Sub(el.lastEventTimes[eventType]) >= el.eventLogCooldown {
		log.Println(msg)
		el.Notify()
		el.lastEventTimes[eventType] = now
	}
}
