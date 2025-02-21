package general

import (
	"fmt"
	"im-service/internal/loadmonitor"
	"log"
	"math/rand"
	"time"
)

// 全局变量，记录上一次强制选择的时间
var lastForcedSelectionTime time.Time

// 强制选择的时间间隔，例如 10 分钟
const forcedSelectionInterval = 10 * time.Minute

// PickServerWithP2C  使用P2C算法选择服务实例
func PickServerWithP2C(endpoints []string, lm *loadmonitor.LoadMonitor) (string, error) {
	log.Printf("使用P2C算法选择服务实例")
	if len(endpoints) == 0 {
		return "", fmt.Errorf("没有可用的端点")
	}
	if len(endpoints) == 1 {
		return endpoints[0], nil
	}

	// 检查是否需要强制选择
	if time.Since(lastForcedSelectionTime) >= forcedSelectionInterval {
		log.Printf("超过一定时间，进行强制选择")
		return pickLowestLoadServer(endpoints, lm), nil
	}

	// 随机选择两个不同的服务实例
	log.Printf("随机选择两个不同的服务实例")
	index1, index2 := rand.Intn(len(endpoints)), rand.Intn(len(endpoints))
	for index2 == index1 {
		index2 = rand.Intn(len(endpoints))
	}

	endpoint1, endpoint2 := endpoints[index1], endpoints[index2]

	// 获取负载信息
	load1, load2 := lm.GetLoad(endpoint1), lm.GetLoad(endpoint2)

	log.Printf("Endpoint1: %s, Load1: %d, Endpoint2: %s, Load2: %d", endpoint1, load1, endpoint2, load2)

	// 选择负载较低的服务实例
	if load1 <= load2 {
		return endpoint1, nil
	}
	return endpoint2, nil
}

// pickLowestLoadServer 选择所有节点中负载最低的节点
func pickLowestLoadServer(endpoints []string, lm *loadmonitor.LoadMonitor) string {
	var lowestLoad int
	var lowestLoadEndpoint string

	for i, endpoint := range endpoints {
		load := lm.GetLoad(endpoint)
		if i == 0 || load < lowestLoad {
			lowestLoad = load
			lowestLoadEndpoint = endpoint
		}
	}

	// 更新最后一次强制选择的时间
	lastForcedSelectionTime = time.Now()

	return lowestLoadEndpoint
}
