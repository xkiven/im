package loadmonitor

import (
	"bytes"
	"encoding/json"
	"github.com/shirou/gopsutil/cpu"
	"log"
	"net/http"
	"sync"
	"time"
)

// LoadMonitor 封装负载监控系统
type LoadMonitor struct {
	loadInfo      map[string]int
	loadInfoMutex sync.RWMutex
	reportURL     string
}

// NewLoadMonitor 创建一个新的负载监控实例
func NewLoadMonitor(reportURL string) *LoadMonitor {
	return &LoadMonitor{
		loadInfo:  make(map[string]int),
		reportURL: reportURL,
	}
}

// Start 启动负载监控系统，包括启动 HTTP 服务器接收负载信息上报和模拟负载上报
func (lm *LoadMonitor) Start(endpoints []string, interval time.Duration) {
	// 启动 HTTP 服务器，接收负载信息上报
	http.HandleFunc("/report_load", lm.handleLoadReport)
	go func() {
		log.Fatal(http.ListenAndServe(":8081", nil))
	}()

	// 服务实例定期上报负载信息
	go lm.realLoadReporting(endpoints, interval)
}

// handleLoadReport 处理服务实例上报的负载信息
func (lm *LoadMonitor) handleLoadReport(w http.ResponseWriter, r *http.Request) {
	var report struct {
		Endpoint string `json:"endpoint"`
		Load     int    `json:"load"`
	}
	err := json.NewDecoder(r.Body).Decode(&report)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	lm.loadInfoMutex.Lock()
	lm.loadInfo[report.Endpoint] = report.Load
	lm.loadInfoMutex.Unlock()

	w.WriteHeader(http.StatusOK)
}

// realLoadReporting 服务实例定期上报实际负载信息
func (lm *LoadMonitor) realLoadReporting(endpoints []string, interval time.Duration) {
	for {
		for _, endpoint := range endpoints {
			load := getActualLoad(endpoint)
			lm.reportLoad(endpoint, load)
		}
		time.Sleep(interval)
	}
}

// getActualLoad 获取服务实例的实际负载信息
// 使用 gopsutil 库获取 CPU 使用率作为负载信息
func getActualLoad(endpoint string) int {
	// 获取 CPU 使用率百分比
	percent, err := cpu.Percent(0, false)
	if err != nil {
		log.Printf("Error getting CPU usage: %v", err)
		return 0
	}
	// 由于 Percent 函数返回的是一个切片，这里取第一个元素（表示整体 CPU 使用率）
	// 将使用率乘以 100 并转换为整数返回
	return int(percent[0])
}

// reportLoad 上报负载信息
func (lm *LoadMonitor) reportLoad(endpoint string, load int) {
	log.Printf("开始上报端点 %s 的负载信息，负载值为: %d", endpoint, load)
	report := struct {
		Endpoint string `json:"endpoint"`
		Load     int    `json:"load"`
	}{
		Endpoint: endpoint,
		Load:     load,
	}
	client := &http.Client{}
	body, err := json.Marshal(report)
	if err != nil {
		log.Printf("负载信息序列化失败: %v", err)
		return
	}
	log.Printf("负载信息序列化成功: %s", string(body))
	req, err := http.NewRequest("POST", lm.reportURL, bytes.NewBuffer(body))
	if err != nil {
		log.Printf("创建请求时出错: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	//log.Printf("请求创建成功: %v", req)
	_, err = client.Do(req)
	if err != nil {
		log.Printf("发送负载报告时出错: %v", err)
		return
	}
}

// GetLoad 获取服务实例的负载信息
func (lm *LoadMonitor) GetLoad(endpoint string) int {
	lm.loadInfoMutex.RLock()
	defer lm.loadInfoMutex.RUnlock()
	load, ok := lm.loadInfo[endpoint]
	if !ok {
		return 0 // 默认负载为 0
	}
	return load
}
