package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
)

// MetricsHandler 用于封装 Prometheus 指标相关操作
type MetricsHandler struct {
	requestCounter *prometheus.CounterVec
}

// NewMetricsHandler 创建一个新的 MetricsHandler 实例
func NewMetricsHandler() *MetricsHandler {
	// 定义一个计数器指标，用于记录 HTTP 请求的总数
	// 这个计数器会根据请求的方法（如 GET、POST 等）和路径（如 /、/api 等）进行分类统计
	requestCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "my_app_http_requests_total",                                 // 指标的名称，需要在 Prometheus 中唯一
			Help: "Total number of HTTP requests received by the application.", // 指标的帮助信息，用于描述指标的含义
		},
		[]string{"method", "path"}, // 标签，用于对指标进行分类
	)

	// 注册指标，只有注册后的指标才能被 Prometheus 采集
	err := prometheus.Register(requestCounter)
	if err != nil {
		log.Printf("Failed to register metrics: %v", err)
		log.Fatalf("Failed to register requestCounter: %v", err)
	}

	return &MetricsHandler{
		requestCounter: requestCounter,
	}
}

// RootHandler 定义一个简单的 HTTP 处理函数，用于处理根路径的请求
func (m *MetricsHandler) RootHandler(w http.ResponseWriter, r *http.Request) {
	// 增加计数器的值，根据请求的方法和路径进行分类
	m.requestCounter.WithLabelValues(r.Method, r.URL.Path).Inc()
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hello, World!"))
}

// RegisterMetricsHandler 注册 Prometheus 指标端点
func (m *MetricsHandler) RegisterMetricsHandler() {
	http.Handle("/metrics", promhttp.Handler())
}
