package target

// Target 表示一个可被健康检查的目标。
type Target interface {
	GetName() string
	Check() error
}
