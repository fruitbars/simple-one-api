package config

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
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
			return &enabledServices[rand.Intn(len(enabledServices))], nil
		default:
			return &enabledServices[rand.Intn(len(enabledServices))], nil
		}
	}
	return nil, fmt.Errorf("model %s not found in the configuration", modelName)
}

func GetRandomEnabledModelDetails() (*ModelDetails, error) {
	// 设置随机数种子
	//rand.Seed(time.Now().UnixNano())

	// 创建一个切片存储所有 Enabled 为 true 的 ModelDetails
	var enabledModels []ModelDetails

	// 遍历 ModelToService 映射，收集所有 Enabled 为 true 的 ModelDetails
	for _, models := range ModelToService {
		for _, model := range models {
			if model.ServiceModel.Enabled {
				enabledModels = append(enabledModels, model)
			}
		}
	}

	// 检查是否有任何 Enabled 为 true 的 ModelDetails
	if len(enabledModels) == 0 {
		return nil, fmt.Errorf("no enabled ModelDetails found")
	}

	// 随机选择一个 Enabled 为 true 的 ModelDetails
	randomModel := enabledModels[rand.Intn(len(enabledModels))]

	return &randomModel, nil
}

func GetRandomEnabledModelDetailsV1() (*ModelDetails, string, error) {
	md, err := GetRandomEnabledModelDetails()
	if err != nil {
		return nil, "", err
	}

	randomString := md.Models[rand.Intn(len(md.Models))]

	log.Println(randomString)

	return md, randomString, nil

}
