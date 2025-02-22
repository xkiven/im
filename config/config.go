package config

import (
	"fmt"
	"github.com/afex/hystrix-go/hystrix"
	"github.com/zeromicro/go-zero/zrpc"
	"gopkg.in/yaml.v3"
	"log"
	"os"
)

type Config struct {
	Name       string             `yaml:"Name"`
	Host       string             `yaml:"Host"`
	Port       int                `yaml:"Port"`
	UserRpc    zrpc.RpcClientConf `yaml:"UserRpc"`
	MessageRpc zrpc.RpcClientConf `yaml:"MessageRpc"`
	FriendRpc  zrpc.RpcClientConf `yaml:"FriendRpc"`
	Kafka      struct {
		Brokers []string `yaml:"Brokers"`
		Topic   string   `yaml:"Topic"`
	} `yaml:"Kafka"`
	MongoDB struct {
		URI      string `yaml:"URI"`
		Database string `yaml:"Database"`
	} `yaml:"MongoDB"`
	MySQL struct {
		DataSource string `yaml:"DataSource"`
	} `yaml:"MySQL"`
	Redis struct {
		Host string `yaml:"Host"`
		Pass string `yaml:"Pass"`
	} `yaml:"Redis"`
}

// 熔断处理函数，当 LoadConfig 失败时调用
func fallbackLoadConfig(filePath string, cfg *Config) error {

	return fmt.Errorf("配置加载失败，启用熔断降级")
}

func init() {
	// 配置熔断命令
	hystrix.ConfigureCommand("load_config", hystrix.CommandConfig{
		Timeout:               1000, // 超时时间，单位毫秒
		MaxConcurrentRequests: 100,  // 最大并发请求数
		ErrorPercentThreshold: 25,   // 错误率阈值，超过该阈值会触发熔断
		SleepWindow:           5000, // 熔断后的休眠窗口时间，单位毫秒
	})
}

// LoadConfig 加载配置文件
func LoadConfig(filePath string, cfg *Config) error {

	err := hystrix.Do("load_config", func() error {
		log.Printf("启用熔断保护")
		// LoadConfig 逻辑
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("读取配置文件失败: %w", err)
		}
		err = yaml.Unmarshal(data, cfg)
		if err != nil {
			return fmt.Errorf("无法重新解组配置文件: %w", err)
		}
		// 手动解析 UserRpc 的 Endpoints
		if len(cfg.UserRpc.Endpoints) == 0 {
			var yamlMap map[string]interface{}
			err = yaml.Unmarshal(data, &yamlMap)
			if err != nil {
				return fmt.Errorf("无法重新解组配置文件: %w", err)
			}
			if userRpc, ok := yamlMap["UserRpc"].(map[string]interface{}); ok {
				if endpoints, ok := userRpc["Endpoints"].([]interface{}); ok {
					for _, endpoint := range endpoints {
						if endpointStr, ok := endpoint.(string); ok {
							cfg.UserRpc.Endpoints = append(cfg.UserRpc.Endpoints, endpointStr)
						}
					}
				}
			}
		}
		// 手动解析 MessageRpc 的 Endpoints
		if len(cfg.MessageRpc.Endpoints) == 0 {
			var yamlMap map[string]interface{}
			err = yaml.Unmarshal(data, &yamlMap)
			if err != nil {
				return fmt.Errorf("无法重新解组配置文件: %w", err)
			}
			if messageRpc, ok := yamlMap["MessageRpc"].(map[string]interface{}); ok {
				if endpoints, ok := messageRpc["Endpoints"].([]interface{}); ok {
					for _, endpoint := range endpoints {
						if endpointStr, ok := endpoint.(string); ok {
							cfg.MessageRpc.Endpoints = append(cfg.MessageRpc.Endpoints, endpointStr)
						}
					}
				}
			}
		}
		// 手动解析 FriendRpc 的 Endpoints
		if len(cfg.FriendRpc.Endpoints) == 0 {
			var yamlMap map[string]interface{}
			err = yaml.Unmarshal(data, &yamlMap)
			if err != nil {
				return fmt.Errorf("无法重新解组配置文件: %w", err)
			}
			if FriendRpc, ok := yamlMap["FriendRpc"].(map[string]interface{}); ok {
				if endpoints, ok := FriendRpc["Endpoints"].([]interface{}); ok {
					for _, endpoint := range endpoints {
						if endpointStr, ok := endpoint.(string); ok {
							cfg.FriendRpc.Endpoints = append(cfg.FriendRpc.Endpoints, endpointStr)
						}
					}
				}
			}
		}
		//fmt.Printf("反序列化配置: %+v\n", cfg)
		return nil
	}, func(err error) error {
		return fallbackLoadConfig(filePath, cfg)
	})

	if err != nil {
		fmt.Println("配置加载失败:", err)
	}

	return nil
}
