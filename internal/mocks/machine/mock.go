package machine

import (
	"sync/atomic"

	"upswatch/internal/machine"
)

type MockMachine struct {
	calls int32
}

var _ machine.MachineInterface = (*MockMachine)(nil)

func New() *MockMachine {
	return &MockMachine{}
}

func (m *MockMachine) Shutdown() {
	atomic.AddInt32(&m.calls, 1)
}

func (m *MockMachine) Calls() int32 {
	return atomic.LoadInt32(&m.calls)
}
