package target

import (
	"fmt"
	"net"
	"time"
)

// TCPTarget 通过 TCP 三次握手检查目标连通性。
type TCPTarget struct {
	Name    string
	Address string
	Timeout time.Duration
}

// TCPOption 用于配置 TCPTarget。
type TCPOption func(*TCPTarget)

// NewTCPTarget 创建 TCPTarget，必要参数为 name、address，其余通过 TCPOption 配置。
func NewTCPTarget(name, address string, opts ...TCPOption) *TCPTarget {
	t := &TCPTarget{
		Name:    name,
		Address: address,
		Timeout: 5 * time.Second,
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

func WithTCPTimeout(timeout time.Duration) TCPOption {
	return func(t *TCPTarget) { t.Timeout = timeout }
}

func (t *TCPTarget) GetName() string {
	return t.Name
}

// Check 执行 TCP 连通性检查。
func (t *TCPTarget) Check() error {
	conn, err := net.DialTimeout("tcp", t.Address, t.Timeout)
	if err != nil {
		return fmt.Errorf("tcp dial %s: %w", t.Address, err)
	}
	conn.Close()
	return nil
}
