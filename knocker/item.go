package knocker

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/LouYuanbo1/go-knocker/alerter"
	"github.com/LouYuanbo1/go-knocker/target"
)

// Status 表示检查目标的状态。
type Status int

const (
	StatusUnknown   Status = iota // 初始状态
	StatusHealthy                 // 健康
	StatusUnhealthy               // 不健康
)

// Item 包含一个检查目标及其关联的告警器。
type Item struct {
	Target     target.Target
	Alerters   []alerter.Alerter
	RetryCount int           // 失败后重试次数，默认 3
	RetryDelay time.Duration // 重试间隔，默认 5s
	status     Status
	mu         sync.Mutex
}

func NewItem(target target.Target, alerters []alerter.Alerter, retryCount int, retryDelay time.Duration) *Item {
	if retryCount <= 0 {
		retryCount = 3
	}
	if retryDelay <= 0 {
		retryDelay = 5 * time.Second
	}
	return &Item{
		Target:     target,
		Alerters:   alerters,
		RetryCount: retryCount,
		RetryDelay: retryDelay,
	}
}

// check 对单个 Item 执行检查并处理状态变化。
// 当检查失败时会进行重试，避免网络抖动导致的误报。
// done 用于接收停止信号，重试期间可被中断。
func (item *Item) check(done <-chan struct{}) {
	err := item.Target.Check()
	if err != nil {
		err = item.retry(done, err)
	}

	item.mu.Lock()
	prevStatus := item.status
	if err != nil {
		item.status = StatusUnhealthy
	} else {
		item.status = StatusHealthy
	}
	currentStatus := item.status
	item.mu.Unlock()

	switch {
	case prevStatus == StatusUnknown && currentStatus == StatusHealthy:
		log.Printf("[%s] 初始检查通过", item.Target.GetName())
	case prevStatus == StatusUnknown && currentStatus == StatusUnhealthy:
		log.Printf("[%s] 初始检查失败: %v", item.Target.GetName(), err)
		item.alert(fmt.Sprintf("[%s] 服务不可用: %v", item.Target.GetName(), err))
	case prevStatus == StatusHealthy && currentStatus == StatusHealthy:
		log.Printf("[%s] 检查通过", item.Target.GetName())
	case prevStatus == StatusHealthy && currentStatus == StatusUnhealthy:
		log.Printf("[%s] 服务异常: %v", item.Target.GetName(), err)
		item.alert(fmt.Sprintf("[%s] 服务异常: %v", item.Target.GetName(), err))
	case prevStatus == StatusUnhealthy && currentStatus == StatusHealthy:
		log.Printf("[%s] 服务已恢复", item.Target.GetName())
		item.alert(fmt.Sprintf("[%s] 服务已恢复", item.Target.GetName()))
	}
}

// retry 按配置的重试次数和间隔重新检查，全部失败返回最后一次错误，任意成功返回 nil。
// 重试期间收到 done 信号会立即中断，返回最后一次错误。
func (item *Item) retry(done <-chan struct{}, lastErr error) error {
	timer := time.NewTimer(0) // 首次立即触发，后续由 Reset 控制延迟
	defer timer.Stop()

	for i := 0; i < item.RetryCount; i++ {
		select {
		case <-done:
			log.Printf("[%s] 重试被中断", item.Target.GetName())
			return lastErr
		case <-timer.C:
		}

		if err := item.Target.Check(); err != nil {
			lastErr = err
			log.Printf("[%s] 重试 %d/%d 失败: %v", item.Target.GetName(), i+1, item.RetryCount, err)
		} else {
			log.Printf("[%s] 重试 %d/%d 成功，已恢复", item.Target.GetName(), i+1, item.RetryCount)
			return nil
		}
		timer.Reset(item.RetryDelay)
	}
	return lastErr
}

// alert 向所有关联的告警器发送通知。
func (item *Item) alert(msg string) {
	for _, a := range item.Alerters {
		if err := a.Alert(msg); err != nil {
			log.Printf("[%s] 告警发送失败 [%s]: %v", item.Target.GetName(), a.GetName(), err)
		}
	}
}
