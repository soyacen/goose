package bbr

import (
	"sync/atomic"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
)

// 全局变量定义
var (
	// cpuUsage 存储CPU使用率的原子值
	// 存储范围：0.0-1.0 (小数格式)
	cpuUsage atomic.Value

	// cpuInterval CPU采样间隔时间
	// 默认值：500毫秒
	// 使用原子操作保证并发安全
	cpuInterval int64 = int64(time.Millisecond * 500)
)

// init 包初始化函数
// 在程序启动时自动执行，初始化CPU监控
func init() {
	// 初始化CPU使用率为0
	cpuUsage.Store(0.0)

	// 启动后台goroutine持续收集CPU使用率
	go collectCPU()
}

// defaultCPU 获取当前CPU使用率
// 返回值范围：0.0-1.0 (小数)
// 线程安全，可并发调用
func defaultCPU() float64 {
	return cpuUsage.Load().(float64)
}

// setCPUInterval 设置CPU采样间隔
// 参数 interval: 采样间隔时间
// 只有当interval大于0时才会更新间隔值
func setCPUInterval(interval time.Duration) {
	if interval > 0 {
		atomic.StoreInt64(&cpuInterval, int64(interval))
	}
}

// collectCPU 持续收集CPU使用率的后台函数
// 这是一个无限循环的goroutine，定期采集CPU数据
func collectCPU() {
	for {
		// 原子加载当前采样间隔
		interval := time.Duration(atomic.LoadInt64(&cpuInterval))

		// Percent返回CPU使用率百分比
		// 使用较小的时间间隔获得有意义的读数
		percentages, err := cpu.Percent(interval, false)

		if err == nil && len(percentages) > 0 {
			// Percent返回0-100范围的值，转换为0-1
			cpuUsage.Store(percentages[0] / 100.0)
		} else {
			// 如果出现错误，等待后再重试
			// 避免紧密循环消耗过多CPU资源
			time.Sleep(interval)
		}
	}
}
