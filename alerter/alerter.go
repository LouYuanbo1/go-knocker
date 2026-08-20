package alerter

// Alerter 告警通知接口。
type Alerter interface {
	GetName() string
	Alert(msg string) error
}
