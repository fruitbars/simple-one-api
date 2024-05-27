package config

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"
)

var ModelToService map[string][]ModelDetails
var LoadBalancingStrategy string
var ServerPort string

// 定义相关结构体
type ServiceModel struct {
	Models      []string          `json:"models"`
	Enabled     bool              `json:"enabled"`
	Credentials map[string]string `json:"credentials"`
	ServerURL   string            `json:"server_url"`
}

type Configuration struct {
	ServerPort    string                    `json:"server_port"`
	LoadBalancing string                    `json:"load_balancing"`
	Services      map[string][]ServiceModel `json:"services"`
}

// ModelDetails 结构用于返回模型相关的服务信息
type ModelDetails struct {
	ServiceName string
	ServiceModel
}

// 创建模型到服务的映射
func createModelToServiceMap(config Configuration) map[string][]ModelDetails {
	modelToService := make(map[string][]ModelDetails)
	for serviceName, serviceModels := range config.Services {
		for _, model := range serviceModels {
			for _, modelName := range model.Models {
				detail := ModelDetails{
					ServiceName:  serviceName,
					ServiceModel: model,
				}
				modelToService[modelName] = append(modelToService[modelName], detail)
			}
		}
	}
	return modelToService
}

// 初始化配置
func InitConfig(configName string) {
	if configName == "" {
		configName = "config.json"
	}
	// 从文件读取配置数据
	data, err := os.ReadFile(configName)
	if err != nil {
		log.Fatalf("Error reading JSON file: %s", err)
	}

	log.Println("read config ok,", configName)

	// 解析 JSON 数据到结构体
	var config Configuration
	err = json.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("Error parsing JSON data: %s", err)
	}

	// 设置负载均衡策略，默认为 "first"
	if config.LoadBalancing == "" {
		LoadBalancingStrategy = "first"
	} else {
		LoadBalancingStrategy = config.LoadBalancing
	}

	log.Println("read LoadBalancingStrategy ok,", LoadBalancingStrategy)

	// 设置服务器端口，默认为 "9090"
	if config.ServerPort == "" {
		ServerPort = ":9090"
	} else {
		ServerPort = config.ServerPort
	}

	log.Println("read ServerPort ok,", ServerPort)
	// 创建映射
	ModelToService = createModelToServiceMap(config)
}

// 根据模型名称获取服务和凭证信息
func GetAllModelService(modelName string) ([]ModelDetails, error) {
	if serviceDetails, found := ModelToService[modelName]; found {
		return serviceDetails, nil
	}
	return nil, fmt.Errorf("model %s not found in the configuration", modelName)
}

// 根据模型名称获取启用的服务和凭证信息
func GetModelService(modelName string) (*ModelDetails, error) {
	if serviceDetails, found := ModelToService[modelName]; found {
		enabledServices := []ModelDetails{}
		for _, sd := range serviceDetails {
			if sd.Enabled {
				enabledServices = append(enabledServices, sd)
			}
		}

		if len(enabledServices) == 0 {
			return nil, fmt.Errorf("no enabled model %s found in the configuration", modelName)
		}

		switch LoadBalancingStrategy {
		case "first":
			return &enabledServices[0], nil
		case "random":
			rand.Seed(time.Now().UnixNano())
			return &enabledServices[rand.Intn(len(enabledServices))], nil
		default:
			return &enabledServices[0], nil
		}
	}
	return nil, fmt.Errorf("model %s not found in the configuration", modelName)
}
