package ups

type UpsInterface interface {
	Connect() error
	Disconnect() error
	IsConnected() bool
	QueryStatus() (UpsStatus, error)
	ToogleBuzzer() error
}
