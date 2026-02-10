package bbr

import (
	"math"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrLimitExceeded 当限流触发时返回的错误
// 表示请求被限流器拒绝
var ErrLimitExceeded = status.Error(codes.ResourceExhausted, "ratelimitgrpc: rate limit exceeded")

// options 限流器配置选项结构体
// 用于配置BBR限流算法的各项参数
type options struct {
	// Window 统计时间窗口
	// 用于计算请求通过率和响应时间的时间范围
	Window time.Duration

	// Buckets 时间窗口内的桶数量
	// 将时间窗口分割成多个桶进行精细化统计
	Buckets int

	// CPUThreshold CPU触发阈值 (0.0-1.0)
	// 当CPU使用率超过此阈值时开始限流
	// 单位为小数，例如0.8表示80%
	CPUThreshold float64

	// CPU 获取当前CPU使用率的函数
	// 返回值范围0.0-1.0
	CPU func() float64

	// CPUInterval CPU采样间隔
	CPUInterval time.Duration
}

// Option 配置选项函数类型
// 用于链式调用设置各种配置参数
type Option func(*options)

// WithWindow 设置统计时间窗口
// 参数 window: 统计时间窗口长度
func WithWindow(window time.Duration) Option {
	return func(o *options) {
		o.Window = window
	}
}

// WithBuckets 设置时间窗口内的桶数量
// 参数 buckets: 时间桶数量，影响统计精度
func WithBuckets(buckets int) Option {
	return func(o *options) {
		o.Buckets = buckets
	}
}

// WithCPUThreshold 设置CPU触发阈值 (0.0-1.0)
// 参数 threshold: CPU使用率阈值，超过此值开始限流
func WithCPUThreshold(threshold float64) Option {
	return func(o *options) {
		o.CPUThreshold = threshold
	}
}

// WithCPU 设置CPU获取函数
// 参数 cpu: 自定义的CPU使用率获取函数
func WithCPU(cpu func() float64) Option {
	return func(o *options) {
		o.CPU = cpu
	}
}

// WithCPUInterval 设置CPU采样间隔
// 参数 interval: CPU采样的时间间隔
func WithCPUInterval(interval time.Duration) Option {
	return func(o *options) {
		o.CPUInterval = interval
	}
}

// defaultOptions 返回默认配置
// 默认值：
// - Window: 10秒
// - Buckets: 100个
// - CPUThreshold: 0.8 (80%)
// - CPU: defaultCPU函数
func defaultOptions() *options {
	return &options{
		Window:       time.Second * 10,
		Buckets:      100,
		CPUThreshold: 0.8,
		CPU:          defaultCPU,
		CPUInterval:  time.Millisecond * 500,
	}
}

// init 初始化配置，确保所有参数有效
// 对未设置或无效的参数使用默认值
func (o *options) init() *options {
	if o.Window <= 0 {
		o.Window = time.Second * 10
	}
	if o.Buckets <= 0 {
		o.Buckets = 100
	}
	if o.CPUThreshold <= 0 {
		o.CPUThreshold = 0.8
	}
	if o.CPUInterval <= 0 {
		o.CPUInterval = time.Millisecond * 500
	}
	if o.CPU == nil {
		o.CPU = defaultCPU
		setCPUInterval(o.CPUInterval)
	}
	return o
}

// apply 应用配置选项
// 按顺序应用所有配置选项函数
func (o *options) apply(opts ...Option) *options {
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// newLimiter 创建新的限流器实例
// 根据配置选项初始化BBR限流器
func (o *options) newLimiter() Limiter {
	return &bbrLimiter{
		conf:     o,
		passStat: newRollingCounter(o.Window, o.Buckets, false),
		rtStat:   newRollingCounter(o.Window, o.Buckets, true),
		cpu:      o.CPU,
	}
}

// DoneInfo 包含请求执行完成的信息
type DoneInfo struct {
	// Status 是响应状态码
	Status int
}

// Limiter 定义限流器接口
type Limiter interface {
	// Allow 检查请求是否被允许
	// 返回完成回调函数和可能的错误
	// 如果请求不被允许，返回错误
	Allow() (done func(DoneInfo), err error)
}

// bbrLimiter BBR限流器的具体实现
// 基于字节跳动BBR算法实现自适应限流
type bbrLimiter struct {
	// mu 读写锁，保护并发访问
	mu sync.RWMutex

	// conf 配置选项引用
	conf *options

	// inflight 当前正在处理的请求数
	inflight int64

	// metrics for the BBR algorithm
	// BBR算法的指标统计
	passStat *rollingCounter // 通过请求数统计
	rtStat   *rollingCounter // 响应时间统计

	// lastDrop 上次丢弃请求的时间
	lastDrop atomic.Pointer[time.Time]

	// cpu 获取当前CPU使用率的函数
	// 返回值范围0.0-1.0
	cpu func() float64
}

// Allow 检查请求是否允许执行
// 返回完成回调函数和可能的错误
func (l *bbrLimiter) Allow() (func(DoneInfo), error) {
	// 检查是否应该丢弃请求
	if l.shouldDrop() {
		return nil, ErrLimitExceeded
	}

	// 增加当前并发请求数
	atomic.AddInt64(&l.inflight, 1)
	startTime := time.Now()

	// 返回完成回调函数
	return func(info DoneInfo) {
		// 确保减少并发请求数，即使发生了异常
		defer atomic.AddInt64(&l.inflight, -1)

		// 计算请求处理时间(毫秒)
		rt := int64(time.Since(startTime) / time.Millisecond)

		// 如果请求成功，更新统计信息
		if info.Status >= 200 && info.Status <= 299 {
			l.passStat.Add(1) // 增加成功请求数
			l.rtStat.Add(rt)  // 记录响应时间
		}
	}, nil
}

// shouldDrop 判断是否应该丢弃请求
// 实现BBR算法的核心决策逻辑
func (l *bbrLimiter) shouldDrop() bool {
	// 检查CPU使用率
	cpu := l.cpu()
	if cpu < l.conf.CPUThreshold {
		// CPU使用率低于阈值
		lastDrop := l.lastDrop.Load()
		if lastDrop == nil {
			// 没有最近的丢弃记录，允许请求
			return false
		}
		// 检查距离上次丢弃是否已经超过1秒
		if time.Since(*lastDrop) > time.Second {
			// 超过1秒，清除丢弃记录，允许请求
			l.lastDrop.Store(nil)
			return false
		}
	}

	// 检查并发请求数
	inflight := atomic.LoadInt64(&l.inflight)
	if inflight <= 1 {
		// 并发数很少，允许请求
		return false
	}

	// 检查是否超过最大允许并发数
	if float64(inflight) > l.maxInflight() {
		// 超过限制，记录丢弃时间并拒绝请求
		now := time.Now()
		l.lastDrop.Store(&now)
		return true
	}

	return false
}

// maxInflight 计算最大允许并发请求数
// 基于BBR算法：max_inflight = max_pass * min_rt
func (l *bbrLimiter) maxInflight() float64 {
	// max_pass * min_rt
	// 使用窗口内的最大通过数和最小平均响应时间

	// 获取窗口内的最大通过请求数
	maxPass := l.passStat.Max()

	// 获取窗口内的最小响应时间
	minRT := l.rtStat.Min()

	// 将桶统计转换为每秒统计
	// maxPass是每个桶的值，需要转换为窗口总量或按比例缩放
	// 实际计算：maxPass * minRt / bucketDurationInSeconds
	// minRT单位是毫秒

	bucketDuration := float64(l.conf.Window) / float64(l.conf.Buckets) / float64(time.Second)
	return float64(maxPass) * float64(minRT) / 1000.0 / bucketDuration
}

// rollingCounter 简化的滑动窗口统计器
// 用于在固定时间窗口内统计各种指标
type rollingCounter struct {
	// mu 读写锁，保证并发安全
	mu sync.RWMutex

	// buckets 时间桶数组
	buckets []int64

	// window 总时间窗口大小
	window time.Duration

	// bucketDur 每个桶的时间长度
	bucketDur time.Duration

	// lastUpdate 上次更新时间
	lastUpdate time.Time

	// isMin 是否跟踪最小值（否则跟踪总和）
	isMin bool
}

// newRollingCounter 创建新的滚动计数器实例
// 参数:
//   - window: 总时间窗口
//   - buckets: 桶数量
//   - isMin: 是否记录每个桶的最小值
func newRollingCounter(window time.Duration, buckets int, isMin bool) *rollingCounter {
	rc := &rollingCounter{
		buckets:    make([]int64, buckets),
		window:     window,
		bucketDur:  window / time.Duration(buckets),
		lastUpdate: time.Now(),
		isMin:      isMin,
	}
	if isMin {
		for i := range rc.buckets {
			rc.buckets[i] = math.MaxInt64
		}
	}
	return rc
}

// Add 向统计器中添加数值
// 参数 val: 要添加的数值
func (c *rollingCounter) Add(val int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 执行桶轮转
	c.rotate()

	// 计算当前桶索引并添加数值
	idx := (time.Now().UnixNano() / int64(c.bucketDur)) % int64(len(c.buckets))
	if c.isMin {
		if val < c.buckets[idx] {
			c.buckets[idx] = val
		}
	} else {
		c.buckets[idx] += val
	}
}

// rotate 执行桶轮转操作
// 清理过期的时间桶数据
func (c *rollingCounter) rotate() {
	now := time.Now()

	// 如果距离上次更新不足一个桶的时间间隔，则无需轮转
	if now.Sub(c.lastUpdate) < c.bucketDur {
		return
	}

	// 计算需要清空的桶数量
	elapsed := now.Sub(c.lastUpdate)
	numToClear := int(elapsed / c.bucketDur)

	// 限制清空数量不超过桶总数
	if numToClear > len(c.buckets) {
		numToClear = len(c.buckets)
	}

	// 计算上次更新时的桶索引
	lastIdx := (c.lastUpdate.UnixNano() / int64(c.bucketDur)) % int64(len(c.buckets))

	// 清空相应数量的旧桶
	for i := 1; i <= numToClear; i++ {
		idx := (lastIdx + int64(i)) % int64(len(c.buckets))
		if c.isMin {
			c.buckets[idx] = math.MaxInt64
		} else {
			c.buckets[idx] = 0
		}
	}

	// 更新最后更新时间
	c.lastUpdate = now
}

// Max 获取所有桶中的最大值
// 用于获取窗口内的峰值统计
func (c *rollingCounter) Max() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var max int64
	for _, v := range c.buckets {
		if !c.isMin {
			if v > max {
				max = v
			}
		} else {
			if v != math.MaxInt64 && v > max {
				max = v
			}
		}
	}
	return max
}

// Min 获取所有桶中的最小正值
// 用于获取窗口内的最佳性能指标
func (c *rollingCounter) Min() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var min int64 = math.MaxInt64
	var found bool
	for _, v := range c.buckets {
		if v > 0 && v != math.MaxInt64 {
			if v < min {
				min = v
			}
			found = true
		}
	}

	// 如果没有找到正值，返回1避免除零错误
	if !found {
		return 1 // Avoid division by zero or nonsensical results
	}
	return min
}
