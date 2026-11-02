package knocker

import "sync"

// Status 表示检查目标的状态。
type Status int

const (
	StatusUnknown   Status = iota // 初始状态
	StatusHealthy                 // 健康
	StatusUnhealthy               // 不健康
)

// trigger 标识一次 from -> to 的状态转移。
type trigger struct{ from, to Status }

// TransitionHandler 状态机内置的状态转移处理器，在对应转移发生时被调用。
// prev/cur 为转移前后状态，err 为本次检查产生的错误（成功时为 nil）。
type TransitionHandler func(prev, cur Status, err error)

// HealthStateMachine 是一个线程安全的状态机，负责保存检查目标的健康状态并执行状态转移。
// 每种状态转移都可经由 On 注册对应的处理器，转移发生后自动以回调方式触发。
type HealthStateMachine struct {
	status   Status
	mu       sync.Mutex
	handlers map[trigger]TransitionHandler // 各 from->to 转移对应的处理器
}

func NewHealthStateMachine() *HealthStateMachine {
	return &HealthStateMachine{
		status:   StatusUnknown,
		handlers: make(map[trigger]TransitionHandler),
	}
}

// On 注册一条 from -> to 状态转移对应的处理器 h。
// 通常在初始化阶段注册；即使运行期并发调用，内部也做了同步保护。
// 返回接收者自身以支持链式注册。
func (sm *HealthStateMachine) On(from, to Status, h TransitionHandler) *HealthStateMachine {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.handlers[trigger{from: from, to: to}] = h
	return sm
}

// OnFirstHealthy 注册"初始检查通过"（Unknown -> Healthy）的处理器。
func (sm *HealthStateMachine) OnFirstHealthy(h TransitionHandler) *HealthStateMachine {
	return sm.On(StatusUnknown, StatusHealthy, h)
}

// OnFirstUnhealthy 注册"初始检查失败"（Unknown -> Unhealthy）的处理器。
func (sm *HealthStateMachine) OnFirstUnhealthy(h TransitionHandler) *HealthStateMachine {
	return sm.On(StatusUnknown, StatusUnhealthy, h)
}

// OnStayHealthy 注册"持续健康"（Healthy -> Healthy）的处理器，即心跳日志。
func (sm *HealthStateMachine) OnStayHealthy(h TransitionHandler) *HealthStateMachine {
	return sm.On(StatusHealthy, StatusHealthy, h)
}

// OnBecomeUnhealthy 注册"转为异常"（Healthy -> Unhealthy）的处理器。
func (sm *HealthStateMachine) OnBecomeUnhealthy(h TransitionHandler) *HealthStateMachine {
	return sm.On(StatusHealthy, StatusUnhealthy, h)
}

// OnRecover 注册"恢复健康"（Unhealthy -> Healthy）的处理器。
func (sm *HealthStateMachine) OnRecover(h TransitionHandler) *HealthStateMachine {
	return sm.On(StatusUnhealthy, StatusHealthy, h)
}

// transit 将状态机转移到新状态 to，并在锁外自动触发该 from -> to 转移注册的处理器。
// 线程安全；即使 to 与当前状态相同，也会按 to -> to 处理（例如持续健康时的心跳日志）。
// err 为本次检查产生的错误（成功时为 nil），会透传给处理器用于日志/告警。
func (sm *HealthStateMachine) transit(to Status, err error) {
	sm.mu.Lock()
	prev := sm.status
	h := sm.handlers[trigger{from: prev, to: to}]
	sm.status = to
	sm.mu.Unlock()

	if h != nil {
		h(prev, to, err)
	}
}

// Checked 根据一次检查结果（err != nil 视为不健康）判定新状态并完成转移。
// 该方法是"检查结果 -> 状态"的唯一入口，把对 Status 枚举的引用都封装在状态机内部，
// 调用方（如 Item.check）无需感知也不应直接使用具体状态值。
func (sm *HealthStateMachine) Checked(err error) {
	to := StatusHealthy
	if err != nil {
		to = StatusUnhealthy
	}
	sm.transit(to, err)
}

// Reset 将状态重置为 StatusUnknown，常用于重建状态机的初始场景。
func (sm *HealthStateMachine) Reset() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.status = StatusUnknown
}
