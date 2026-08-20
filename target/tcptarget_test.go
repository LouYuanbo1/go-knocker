package target

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewTCPTarget(t *testing.T) {
	target := NewTCPTarget("mysql", "127.0.0.1:3306")
	assert.Equal(t, "mysql", target.Name)
	assert.Equal(t, "127.0.0.1:3306", target.Address)
	assert.Equal(t, 5*time.Second, target.Timeout)
}

func TestNewTCPTarget_CustomTimeout(t *testing.T) {
	target := NewTCPTarget("redis", "127.0.0.1:6379", WithTCPTimeout(2*time.Second))
	assert.Equal(t, 2*time.Second, target.Timeout)
}

func TestTCPTarget_GetName(t *testing.T) {
	target := NewTCPTarget("kafka", "127.0.0.1:9092")
	assert.Equal(t, "kafka", target.GetName())
}

func TestTCPTarget_Check_Success(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer listener.Close()

	addr := listener.Addr().String()
	target := NewTCPTarget("test", addr, WithTCPTimeout(1*time.Second))
	err = target.Check()
	assert.NoError(t, err)
}

func TestTCPTarget_Check_ConnectionRefused(t *testing.T) {
	target := NewTCPTarget("test", "127.0.0.1:1", WithTCPTimeout(500*time.Millisecond))
	err := target.Check()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tcp dial")
}

func TestTCPTarget_Check_Timeout(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)

	//addr := listener.Addr().String()

	// 关闭 listener 并立即 accept 不存在的连接，制造超时
	listener.Close()

	// 连接到一个不响应 SYN-ACK 的地址（不可路由的 IP）
	target := NewTCPTarget("test", "10.255.255.1:12345", WithTCPTimeout(100*time.Millisecond))
	err = target.Check()
	assert.Error(t, err)
}
