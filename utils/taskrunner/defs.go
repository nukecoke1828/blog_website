package taskrunner

const (
	// 控制通道（controlChan）消息类型常量
	// 这些字符串用于在任务运行器的控制通道中传递指令

	// READY_TO_DISPATCH 表示任务已准备好可以分发
	// "d" 代表 dispatch（分发）
	READY_TO_DISPATCH = "d"

	// READY_TO_EXECUTE 表示任务已准备好可以执行
	// "e" 代表 execute（执行）
	READY_TO_EXECUTE = "e"

	// CLOSE 表示关闭任务或通道
	// "c" 代表 close（关闭）
	CLOSE = "c"
)

// controlChan 控制通道类型定义
// 用于在任务运行器组件间传递控制指令的通道
// 只能传输字符串类型的数据（使用上面定义的常量）
type controlChan chan string

// dataChan 数据通道类型定义
// 用于在任务运行器组件间传递实际的任务数据
// 可以传输任意类型的接口（interface{}），提供了极大的灵活性
type DataChan chan interface{}

// fn 函数类型定义
// 用于定义dispatcher（分发器）和executor（执行器）的函数签名
// 参数：dataChan - 数据通道，用于接收或发送数据
// 返回值：error - 执行过程中可能出现的错误
type fn func(dc DataChan) error
