package taskrunner

import (
	"log"
)

// Runner 任务运行器结构体
// 负责协调任务的分发和执行，管理任务的生命周期
type Runner struct {
	Controller controlChan // 控制通道，用于接收控制指令
	Error      controlChan // 错误通道，用于接收错误信号
	Data       DataChan    // 数据通道，用于传输任务数据
	dataSize   int         // 数据通道的缓冲区大小
	longLived  bool        // 是否长期运行（决定是否关闭通道）
	Dispatcher fn          // 分发器函数，负责分发任务
	Executor   fn          // 执行器函数，负责执行任务
}

// NewRunner 创建新的任务运行器实例
// size: 数据通道的缓冲区大小
// longLived: 是否长期运行（true不自动关闭通道，false自动关闭）
// dispatcher: 任务分发函数
// executor: 任务执行函数
// 返回: 初始化好的Runner指针
func NewRunner(size int, longLived bool, dispatcher fn, executor fn) *Runner {
	return &Runner{
		Controller: make(controlChan, 1), // 带1个缓冲的控制通道
		Error:      make(controlChan, 1), // 带1个缓冲的错误通道
		Data:       make(DataChan, size), // 带指定大小缓冲的数据通道
		dataSize:   size,
		longLived:  longLived,
		Dispatcher: dispatcher,
		Executor:   executor,
	}
}

// startDispatcher 启动任务调度器（核心调度循环）
// 该方法在一个独立的goroutine中运行，负责协调任务流程
func (r *Runner) startDispatcher() {
	// 函数退出时，如果不是长期运行则关闭所有通道
	defer func() {
		if !r.longLived {
			close(r.Controller)
			close(r.Error)
			close(r.Data)
		}
	}()

	// 无限循环，直到收到关闭信号
	for {
		select {
		case c := <-r.Controller: // 监听控制通道
			if c == READY_TO_DISPATCH {
				// 收到分发指令，调用分发器函数
				err := r.Dispatcher(r.Data)
				log.Printf("startDispatch Controller add data: %v\n", r.Data)
				if err != nil {
					// 分发失败，发送关闭信号
					r.Error <- CLOSE
				} else {
					// 分发成功，准备执行
					r.Controller <- READY_TO_EXECUTE
				}
			}

			if c == READY_TO_EXECUTE {
				// 收到执行指令，调用执行器函数
				err := r.Executor(r.Data)
				log.Printf("startDispatch Controller execute data: %v\n", r.Data)
				if err != nil {
					// 执行失败，发送关闭信号
					r.Error <- CLOSE
				} else {
					// 执行成功，准备下一次分发
					r.Controller <- READY_TO_DISPATCH
				}
			}

		case e := <-r.Error: // 监听错误通道
			if e == CLOSE {
				// 收到关闭信号，退出调度循环
				return
			}
		}
	}
}

// StartAll 启动整个任务运行器
// 发送初始的分发指令，并启动调度器
func (r *Runner) StartAll() {
	// 发送初始控制信号，启动任务流程
	r.Controller <- READY_TO_DISPATCH
	// 启动调度器（通常在单独的goroutine中运行）
	r.startDispatcher()
}
