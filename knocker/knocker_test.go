package knocker

import (
	"errors"
	"testing"
	"time"

	"github.com/LouYuanbo1/go-knocker/alerter"
	"github.com/stretchr/testify/assert"
)

func TestNewKnocker(t *testing.T) {
	items := []*Item{}
	k := NewKnocker(items, 10*time.Second)
	assert.Equal(t, items, k.Items)
	assert.Equal(t, 10*time.Second, k.Interval)
	assert.NotNil(t, k.done)
}

func TestNewKnocker_ZeroIntervalDefaults(t *testing.T) {
	k := NewKnocker(nil, 0)
	assert.Equal(t, 30*time.Second, k.Interval)
}

func TestNewKnocker_NegativeIntervalDefaults(t *testing.T) {
	k := NewKnocker(nil, -1*time.Second)
	assert.Equal(t, 30*time.Second, k.Interval)
}

func TestKnocker_RunOnce(t *testing.T) {
	target := &mockTarget{name: "test", checkFn: func() error { return nil }}
	item := NewItem(target, nil, 0, 0)
	k := NewKnocker([]*Item{item}, 10*time.Second)

	k.RunOnce()
	// Should not panic and should complete
}

func TestKnocker_RunOnce_WithError(t *testing.T) {
	target := &mockTarget{name: "test", checkFn: func() error { return errors.New("fail") }}
	a := &mockAlerter{name: "alert"}
	item := NewItem(target, []alerter.Alerter{a}, 0, 0)
	k := NewKnocker([]*Item{item}, 10*time.Second)

	k.RunOnce()
	assert.Len(t, a.alerts, 1)
}

func TestKnocker_Stop(t *testing.T) {
	target := &mockTarget{name: "test", checkFn: func() error {
		time.Sleep(50 * time.Millisecond)
		return nil
	}}
	item := NewItem(target, nil, 0, 0)
	k := NewKnocker([]*Item{item}, 10*time.Millisecond)

	go func() {
		time.Sleep(50 * time.Millisecond)
		k.Stop()
	}()

	k.Run()
	// Should exit cleanly
}

func TestKnocker_Stop_BeforeRun(t *testing.T) {
	k := NewKnocker(nil, 10*time.Second)
	k.Stop()

	// Run should exit immediately since done is already closed
	k.Run()
}

func TestKnocker_RunOnce_MultipleItems(t *testing.T) {
	t1 := &mockTarget{name: "target1", checkFn: func() error { return nil }}
	t2 := &mockTarget{name: "target2", checkFn: func() error { return errors.New("fail") }}
	a1 := &mockAlerter{name: "alert1"}
	a2 := &mockAlerter{name: "alert2"}

	item1 := NewItem(t1, []alerter.Alerter{a1}, 0, 0)
	item2 := NewItem(t2, []alerter.Alerter{a2}, 0, 0)
	k := NewKnocker([]*Item{item1, item2}, 10*time.Second)

	k.RunOnce()

	assert.Empty(t, a1.alerts, "healthy target should not alert")
	assert.Len(t, a2.alerts, 1, "unhealthy target should alert")
}
