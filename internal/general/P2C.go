package general

import (
	"fmt"
	"im-service/internal/loadmonitor"
	"log"
	"math/rand"
)

// PickServerWithP2C  使用P2C算法选择服务实例
func PickServerWithP2C(endpoints []string, lm *loadmonitor.LoadMonitor) (string, error) {
	log.Printf("使用P2C算法选择服务实例")
	if len(endpoints) == 0 {
		return "", fmt.Errorf("no endpoints available")
	}
	if len(endpoints) == 1 {
		return endpoints[0], nil
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
