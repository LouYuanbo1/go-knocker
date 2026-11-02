package knocker

import (
	"fmt"
	"log"
	"time"

	"github.com/LouYuanbo1/go-knocker/alerter"
	"github.com/LouYuanbo1/go-knocker/target"
)

// Item 包含一个检查目标及其关联的告警器。
type Item struct {
	Target     target.Target
	Alerters   []alerter.Alerter
	RetryCount int                 // 失败后重试次数；0 表示不重试
	RetryDelay time.Duration       // 重试间隔；零值表示每次重试立即执行
	state      *HealthStateMachine // 状态机，保存当前状态并处理状态转移
}

func NewItem(target target.Target, alerters []alerter.Alerter, retryCount int, retryDelay time.Duration) *Item {
	item := &Item{
		Target:     target,
		Alerters:   alerters,
		RetryCount: retryCount,
		RetryDelay: retryDelay,
		state:      NewHealthStateMachine(),
	}
	item.registerTransitions()
	return item
}

// registerTransitions 把"状态转移 -> 日志/告警"的响应逻辑内置到状态机。
// 状态变更时由状态机自动触发对应的处理器，无需在 check 中编写冗长的 switch。
func (item *Item) registerTransitions() {
	item.state.
		OnFirstHealthy(func(_, _ Status, _ error) {
			log.Printf("[%s] 初始检查通过", item.Target.GetName())
		}).
		OnFirstUnhealthy(func(_, _ Status, err error) {
			name := item.Target.GetName()
			log.Printf("[%s] 初始检查失败: %v", name, err)
			item.alert(fmt.Sprintf("[%s] 服务不可用: %v", name, err))
		}).
		OnStayHealthy(func(_, _ Status, _ error) {
			log.Printf("[%s] 检查通过", item.Target.GetName())
		}).
		OnBecomeUnhealthy(func(_, _ Status, err error) {
			name := item.Target.GetName()
			log.Printf("[%s] 服务异常: %v", name, err)
			item.alert(fmt.Sprintf("[%s] 服务异常: %v", name, err))
		}).
		OnRecover(func(_, _ Status, _ error) {
			name := item.Target.GetName()
			log.Printf("[%s] 服务已恢复", name)
			item.alert(fmt.Sprintf("[%s] 服务已恢复", name))
		})
}

// check 对单个 Item 执行检查并处理状态变化。
// 当检查失败时会进行重试，避免网络抖动导致的误报。
// done 用于接收停止信号，重试期间可被中断。
func (item *Item) check(done <-chan struct{}) {
	err := item.Target.Check()
	if err != nil {
		err = item.retry(done, err)
	}
	// 状态判定与转移都由状态机内部完成，check 不感知具体状态枚举。
	item.state.Checked(err)
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
