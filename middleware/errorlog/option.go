// Package errorlog 提供HTTP错误日志记录中间件功能
// 用于记录发生错误的HTTP请求，支持配置是否打印请求和响应
package errorlog

// options 保存错误日志中间件的配置选项
type options struct {
	printRequest  bool // 是否打印请求
	printResponse bool // 是否打印响应
}

// Option 是配置错误日志中间件的函数类型
type Option func(o *options)

// defaultOptions 返回默认配置选项
// 返回值:
//   - *options: 默认配置
func defaultOptions() *options {
	return &options{
		printRequest:  false,
		printResponse: false,
	}
}

// apply 应用配置选项
// 参数:
//   - opts: 配置选项列表
//
// 返回值:
//   - *options: 应用配置后的选项
func (o *options) apply(opts ...Option) *options {
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// WithPrintRequest 设置是否打印请求
// 参数:
//   - print: 是否打印请求
//
// 返回值:
//   - Option: 配置选项
func WithPrintRequest(print bool) Option {
	return func(o *options) {
		o.printRequest = print
	}
}

// WithPrintResponse 设置是否打印响应
// 参数:
//   - print: 是否打印响应
//
// 返回值:
//   - Option: 配置选项
func WithPrintResponse(print bool) Option {
	return func(o *options) {
		o.printResponse = print
	}
}
