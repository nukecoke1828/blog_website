package taskrunner

import (
	"log"
	"time"
)

// Worker 工作者结构体，用于定时执行任务
// 封装了定时器和任务运行器，实现周期性的任务调度
type Worker struct {
	ticket *time.Ticker // 定时器，用于定期触发任务执行
	runner *Runner      // 任务运行器，负责具体的任务分发和执行
}

// NewWorker 创建新的工作者实例
// interval: 任务执行的时间间隔（如3秒、5分钟等）
// r: 任务运行器实例，包含具体的任务处理逻辑
// 返回: 初始化好的Worker指针
func NewWorker(interval time.Duration, r *Runner) *Worker {
	return &Worker{
		ticket: time.NewTicker(interval), // 创建定时器，按指定间隔触发
		runner: r,                        // 保存任务运行器引用
	}
}

// startWorker 启动工作者，开始定时执行任务
// 该方法会阻塞当前goroutine，通常需要在单独的goroutine中运行
func (w *Worker) startWorker() {
	// 使用range遍历定时器的通道，每次定时器触发时执行循环体
	for range w.ticket.C {
		log.Printf("ticket run start--------------------\n")

		// 在新的goroutine中启动任务运行器，避免阻塞定时循环
		// 这样即使任务执行时间较长，也不会影响下一次定时触发
		go w.runner.StartAll()

		log.Printf("ticket run end--------------------\n")
	}
	// 定时器停止后，循环会自动退出
}

func Start() {
	// 创建任务运行器实例
	// 参数说明：
	// 3 - 数据通道缓冲区大小
	// true - 长期运行（不自动关闭通道）
	// JWTkeyUpdateDispatcher - 任务分发器，用于分发JWT密钥更新任务
	// JWTkeyUpdateExecutor - 任务执行器，用于执行JWT密钥更新任务
	r := NewRunner(3, true, JWTkeyUpdateDispatcher, JWTkeyUpdateExecutor)

	// 创建工作者实例，设置30天的执行间隔
	w := NewWorker(30*24*time.Hour, r)

	// 在新的goroutine中启动工作者，使其在后台运行
	go w.startWorker()

	// 函数立即返回，工作者在后台持续运行
}
