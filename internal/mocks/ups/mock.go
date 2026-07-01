package ups

import (
	"sync"
	"sync/atomic"

	"upswatch/internal/ups"
)

type MockUps struct {
	mu sync.Mutex

	ConnectErrs []error
	connectIdx  int
	connected   bool

	QueryStatusVal ups.UpsStatus
	QueryErr       error
	QueryFunc      func(call int32) (ups.UpsStatus, error)

	ToggleErr error

	ConnectCalls    int32
	DisconnectCalls int32
	QueryCalls      int32
	ToggleCalls     int32
}

var _ ups.UpsInterface = (*MockUps)(nil)

func New() *MockUps {
	return &MockUps{}
}

func (m *MockUps) Connect() error {
	atomic.AddInt32(&m.ConnectCalls, 1)
	m.mu.Lock()
	defer m.mu.Unlock()
	var err error
	if m.connectIdx < len(m.ConnectErrs) {
		err = m.ConnectErrs[m.connectIdx]
		m.connectIdx++
	}
	if err == nil {
		m.connected = true
	}
	return err
}

func (m *MockUps) Disconnect() error {
	atomic.AddInt32(&m.DisconnectCalls, 1)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = false
	return nil
}

func (m *MockUps) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}

func (m *MockUps) QueryStatus() (ups.UpsStatus, error) {
	call := atomic.AddInt32(&m.QueryCalls, 1)
	m.mu.Lock()
	qf, qs, qe := m.QueryFunc, m.QueryStatusVal, m.QueryErr
	m.mu.Unlock()
	if qf != nil {
		return qf(call)
	}
	return qs, qe
}

func (m *MockUps) ToogleBuzzer() error {
	atomic.AddInt32(&m.ToggleCalls, 1)
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ToggleErr
}

func (m *MockUps) SetConnected(v bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = v
}

func (m *MockUps) SetStatus(s ups.UpsStatus, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.QueryStatusVal = s
	m.QueryErr = err
}
