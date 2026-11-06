package graceful

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Stopper 管理优雅退出的结构体
type Stopper struct {
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	timeout    time.Duration
	finalizers []func() error
	sigChan    chan os.Signal
}

// NewStopper 创建一个新的 Stopper
func NewStopper(timeout_opt ...time.Duration) *Stopper {
	timeout := 3 * time.Second

	if len(timeout_opt) > 0 {
		timeout = timeout_opt[0]
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &Stopper{
		ctx:        ctx,
		cancel:     cancel,
		timeout:    timeout,
		sigChan:    make(chan os.Signal, 1),
		finalizers: make([]func() error, 0),
	}
}

// Stop 停止 Stopper
func (sm *Stopper) Stop() {
	sm.cancel()
}

// IsStopped 检查 Stopper 是否已停止
func (sm *Stopper) IsStopped() bool {
	select {
	case <-sm.ctx.Done():
		return true
	default:
		return false
	}
}

// AddFinalizer 添加退出时需要执行的函数
func (sm *Stopper) AddFinalizer(finalizer ...func() error) {
	sm.finalizers = append(sm.finalizers, finalizer...)
}

// taskError 用于安全地处理任务错误
type taskError struct {
	err error
	mu  sync.Mutex
}

// setError 安全地设置错误，只在第一次设置时生效
func (te *taskError) setError(err error) {
	te.mu.Lock()
	if te.err == nil { // 只在未设置错误时写入
		te.err = err
	}
	te.mu.Unlock()
}

// getError 安全地获取错误
func (te *taskError) getError() error {
	te.mu.Lock()
	defer te.mu.Unlock()
	return te.err
}

// RunMany 运行多个带优雅退出的任务
func (sm *Stopper) RunMany(tasks ...func(ctx context.Context) error) error {
	// 监听系统信号
	signal.Notify(sm.sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sm.sigChan)

	// 用于存储任务执行的错误
	te := &taskError{}

	// 启动所有任务
	sm.wg.Add(len(tasks))
	for i, task := range tasks {
		go func(taskIndex int, t func(ctx context.Context) error) {
			defer sm.wg.Done()
			if err := t(sm.ctx); err != nil {
				log.Printf("任务 %d 发生错误: %v", taskIndex, err)
				te.setError(err) // 记录第一个发生的错误
				sm.cancel()      // 任一任务出错，取消所有任务
			}
		}(i, task)
	}
	// 等待所有任务结束
	doneChan := make(chan struct{})
	go func() {
		sm.wg.Wait()
		close(doneChan)
	}()

	var timeoutCtx context.Context
	var timeoutCancel context.CancelFunc
	// 等待信号 或内部提前退出
	select {
	case sig := <-sm.sigChan:
		log.Printf("收到信号: %v, 开始退出...", sig)
	case <-sm.ctx.Done():
	case <-doneChan:
		log.Println("提前退出")
		goto end
	}

	// 触发取消信号
	sm.cancel()

	// 创建一个带超时的 context
	timeoutCtx, timeoutCancel = context.WithTimeout(context.Background(), sm.timeout)
	defer timeoutCancel()

	// 等待任务完成或超时
	select {
	case <-doneChan:
	case <-timeoutCtx.Done():
		log.Println("任务退出超时")
		te.setError(context.DeadlineExceeded)
	}

end:
	// 逆序执行退出钩子
	for i := len(sm.finalizers) - 1; i >= 0; i-- {
		finalizer := sm.finalizers[i]
		if err := finalizer(); err != nil {
			log.Printf("执行退出钩子时发生错误: %v", err)
			return err
		}
	}

	return te.getError()
}

// Run 运行单个带优雅退出的任务（为了向后兼容）
func (sm *Stopper) Run(task func(ctx context.Context) error) error {
	return sm.RunMany(task)
}

var (
	defaultStopper = NewStopper(3 * time.Second) // 改为小写，包私有
)

// SetTimeout 设置默认 Stopper 的超时时间
func SetTimeout(timeout time.Duration) {
	defaultStopper.timeout = timeout
}

// AddFinalizer 添加退出时需要执行的函数到默认 Stopper
func AddFinalizer(finalizer ...func() error) {
	defaultStopper.AddFinalizer(finalizer...)
}

// RunMany 使用默认 Stopper 运行多个带优雅退出的任务
func RunMany(tasks ...func(ctx context.Context) error) error {
	return defaultStopper.RunMany(tasks...)
}

// Run 使用默认 Stopper 运行单个带优雅退出的任务
func Run(task func(ctx context.Context) error) error {
	return defaultStopper.Run(task)
}

// Stop 停止默认 Stopper
func Stop() {
	defaultStopper.Stop()
}

// IsStopped 检查默认 Stopper 是否已停止
func IsStopped() bool {
	return defaultStopper.IsStopped()
}

// IsTimeoutError 判断错误是否为超时错误
func IsTimeoutError(err error) bool {
	return err == context.DeadlineExceeded
}
