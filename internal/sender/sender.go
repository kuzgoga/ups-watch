package notification

import (
	"log/slog"
)

type SendlerInterface interface {
	Send(level, msg string) error
}

type MultiSender struct {
	senders []SendlerInterface
}

func NewMultiSender(senders ...SendlerInterface) *MultiSender {
	return &MultiSender{senders: senders}
}

func (m *MultiSender) Send(level, msg string) error {
	for _, sender := range m.senders {
		if err := sender.Send(level, msg); err != nil {
			slog.Error("Failed to send notifications", "error", err)
		}
	}
	return nil
}
