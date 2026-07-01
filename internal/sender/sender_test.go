package notification

import "testing"

type MockSender struct {
	Messages []string
}

func (m *MockSender) Send(level, msg string) error {
	m.Messages = append(m.Messages, level+":"+msg)
	return nil
}

func TestMultiSender_Send(t *testing.T) {
	m1 := &MockSender{}
	m2 := &MockSender{}
	multi := NewMultiSender(m1, m2)

	err := multi.Send("warning", "Power lost")
	if err != nil {
		t.Fatal(err)
	}

	if len(m1.Messages) != 1 || m1.Messages[0] != "warning:Power lost" {
		t.Error("M1 didn't received message")
	}
	if len(m2.Messages) != 1 || m2.Messages[0] != "warning:Power lost" {
		t.Error("M2 didn't received message")
	}
}
