package general

import "github.com/afex/hystrix-go/hystrix"

// WithHystrix 是一个通用的熔断保护函数
// commandName 是熔断命令的名称
// run 是正常执行的函数
// fallback 是熔断时执行的函数
func WithHystrix(commandName string, run func() error, fallback func(error) error) error {
	// 配置熔断参数
	hystrix.ConfigureCommand(commandName, hystrix.CommandConfig{
		Timeout:               2000, // 超时时间，单位毫秒
		MaxConcurrentRequests: 100,  // 最大并发请求数
		ErrorPercentThreshold: 25,   // 错误率阈值，超过该阈值会触发熔断
		SleepWindow:           5000, // 熔断后的休眠窗口时间，单位毫秒
	})

	var err error
	// 执行带有熔断保护的操作
	err = hystrix.Do(commandName, func() error {
		return run()
	}, func(err error) error {
		return fallback(err)
	})

	return err
}
