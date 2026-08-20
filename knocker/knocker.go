package knocker

import (
	"sync"
	"time"
)

// Knocker 定时对所有 Item 执行健康检查，并在状态变化时触发告警。
type Knocker struct {
	Items    []*Item
	Interval time.Duration
	done     chan struct{}
}

func NewKnocker(items []*Item, interval time.Duration) *Knocker {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &Knocker{Items: items, Interval: interval, done: make(chan struct{})}
}

// RunOnce 对所有 Item 执行一次检查，状态变化时自动告警。
func (k *Knocker) RunOnce() {
	var wg sync.WaitGroup
	for _, item := range k.Items {
		wg.Go(func() { item.check(k.done) })
	}
	wg.Wait()
}

// Run 阻塞运行，每隔 Interval 执行一次 RunOnce。
// 调用 Stop() 可优雅退出。
func (k *Knocker) Run() {
	ticker := time.NewTicker(k.Interval)
	defer ticker.Stop()

	k.RunOnce()
	for {
		select {
		case <-ticker.C:
			k.RunOnce()
		case <-k.done:
			return
		}
	}
}

// Stop 优雅停止 Run()，等待当前正在执行的检查完成后退出。
func (k *Knocker) Stop() {
	close(k.done)
}
