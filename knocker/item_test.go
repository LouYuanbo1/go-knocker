package knocker

import (
	"errors"
	"testing"
	"time"

	"github.com/LouYuanbo1/go-knocker/alerter"
	"github.com/stretchr/testify/assert"
)

type mockTarget struct {
	name    string
	checkFn func() error
}

func (m *mockTarget) GetName() string { return m.name }
func (m *mockTarget) Check() error    { return m.checkFn() }

type mockAlerter struct {
	name    string
	alertFn func(msg string) error
	alerts  []string
}

func (m *mockAlerter) GetName() string { return m.name }
func (m *mockAlerter) Alert(msg string) error {
	m.alerts = append(m.alerts, msg)
	if m.alertFn != nil {
		return m.alertFn(msg)
	}
	return nil
}

func TestNewItem_Defaults(t *testing.T) {
	target := &mockTarget{name: "test"}
	item := NewItem(target, nil, 0, 0)
	assert.Equal(t, target, item.Target)
	assert.Equal(t, 3, item.RetryCount)
	assert.Equal(t, 5*time.Second, item.RetryDelay)
	assert.Nil(t, item.Alerters)
}

func TestNewItem_CustomRetry(t *testing.T) {
	target := &mockTarget{name: "test"}
	item := NewItem(target, nil, 5, 2*time.Second)
	assert.Equal(t, 5, item.RetryCount)
	assert.Equal(t, 2*time.Second, item.RetryDelay)
}

func TestItem_Check_HealthyFirstTime(t *testing.T) {
	target := &mockTarget{name: "test", checkFn: func() error { return nil }}
	a := &mockAlerter{name: "alert"}
	item := NewItem(target, []alerter.Alerter{a}, 0, 0)

	done := make(chan struct{})
	item.check(done)

	assert.Empty(t, a.alerts, "should not alert on initial healthy")
}

func TestItem_Check_UnhealthyFirstTime_Alerts(t *testing.T) {
	target := &mockTarget{name: "test", checkFn: func() error { return errors.New("fail") }}
	a := &mockAlerter{name: "alert"}
	item := NewItem(target, []alerter.Alerter{a}, 0, 0)

	done := make(chan struct{})
	item.check(done)

	assert.Len(t, a.alerts, 1)
	assert.Contains(t, a.alerts[0], "服务不可用")
}

func TestItem_Check_HealthyToUnhealthy(t *testing.T) {
	healthy := true
	target := &mockTarget{name: "test", checkFn: func() error {
		if healthy {
			return nil
		}
		return errors.New("fail")
	}}
	a := &mockAlerter{name: "alert"}
	item := NewItem(target, []alerter.Alerter{a}, 0, 0)

	done := make(chan struct{})

	// First check: healthy
	item.check(done)
	assert.Empty(t, a.alerts)

	// Second check: unhealthy
	healthy = false
	item.check(done)
	assert.Len(t, a.alerts, 1)
	assert.Contains(t, a.alerts[0], "服务异常")
}

func TestItem_Check_UnhealthyToHealthy(t *testing.T) {
	healthy := false
	target := &mockTarget{name: "test", checkFn: func() error {
		if healthy {
			return nil
		}
		return errors.New("fail")
	}}
	a := &mockAlerter{name: "alert"}
	item := NewItem(target, []alerter.Alerter{a}, 0, 0)

	done := make(chan struct{})

	// First check: unhealthy
	item.check(done)
	assert.Len(t, a.alerts, 1)
	assert.Contains(t, a.alerts[0], "服务不可用")

	// Second check: healthy
	healthy = true
	item.check(done)
	assert.Len(t, a.alerts, 2)
	assert.Contains(t, a.alerts[1], "服务已恢复")
}

func TestItem_Check_UnhealthyStaysUnhealthy_NoRepeatAlert(t *testing.T) {
	target := &mockTarget{name: "test", checkFn: func() error { return errors.New("fail") }}
	a := &mockAlerter{name: "alert"}
	item := NewItem(target, []alerter.Alerter{a}, 0, 0)

	done := make(chan struct{})

	// First check: alerts
	item.check(done)
	assert.Len(t, a.alerts, 1)

	// Second check: no alert (stays unhealthy)
	item.check(done)
	assert.Len(t, a.alerts, 1, "should not repeat alert when already unhealthy")
}

func TestItem_Check_HealthyStaysHealthy_NoAlert(t *testing.T) {
	target := &mockTarget{name: "test", checkFn: func() error { return nil }}
	a := &mockAlerter{name: "alert"}
	item := NewItem(target, []alerter.Alerter{a}, 0, 0)

	done := make(chan struct{})

	item.check(done)
	assert.Empty(t, a.alerts)

	item.check(done)
	assert.Empty(t, a.alerts, "should not alert when already healthy")
}

func TestItem_Check_RetryThenSuccess(t *testing.T) {
	attempts := 0
	target := &mockTarget{name: "test", checkFn: func() error {
		attempts++
		if attempts < 3 {
			return errors.New("fail")
		}
		return nil
	}}
	a := &mockAlerter{name: "alert"}
	item := NewItem(target, []alerter.Alerter{a}, 3, 10*time.Millisecond)

	done := make(chan struct{})
	item.check(done)

	assert.Empty(t, a.alerts, "should not alert because retry succeeded")
	assert.GreaterOrEqual(t, attempts, 3)
}

func TestItem_Check_RetryAllFail_Alerts(t *testing.T) {
	target := &mockTarget{name: "test", checkFn: func() error { return errors.New("fail") }}
	a := &mockAlerter{name: "alert"}
	item := NewItem(target, []alerter.Alerter{a}, 3, 10*time.Millisecond)

	done := make(chan struct{})
	item.check(done)

	assert.Len(t, a.alerts, 1)
	assert.Contains(t, a.alerts[0], "服务不可用")
}

func TestItem_Check_RetryInterruptedByDone(t *testing.T) {
	target := &mockTarget{name: "test", checkFn: func() error { return errors.New("fail") }}
	a := &mockAlerter{name: "alert"}
	item := NewItem(target, []alerter.Alerter{a}, 3, 100*time.Millisecond)

	done := make(chan struct{})
	close(done) // immediately signal stop

	item.check(done)

	assert.Len(t, a.alerts, 1, "should still alert with last error")
}

func TestItem_Check_AlertFails(t *testing.T) {
	target := &mockTarget{name: "test", checkFn: func() error { return errors.New("fail") }}
	a := &mockAlerter{name: "alert", alertFn: func(msg string) error {
		return errors.New("send failed")
	}}
	item := NewItem(target, []alerter.Alerter{a}, 0, 0)

	done := make(chan struct{})
	item.check(done)

	assert.Len(t, a.alerts, 1, "alerter should still receive the message")
}

func TestItem_Check_MultipleAlerters(t *testing.T) {
	target := &mockTarget{name: "test", checkFn: func() error { return errors.New("fail") }}
	a1 := &mockAlerter{name: "alert1"}
	a2 := &mockAlerter{name: "alert2"}
	item := NewItem(target, []alerter.Alerter{a1, a2}, 0, 0)

	done := make(chan struct{})
	item.check(done)

	assert.Len(t, a1.alerts, 1)
	assert.Len(t, a2.alerts, 1)
}
