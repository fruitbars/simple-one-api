package config

import (
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"log"
	"math/rand"
	"os"
	"simple-one-api/pkg/mylog"
	"simple-one-api/pkg/utils"
	"sort"
	"strings"
	"time"
)

var defaultLimitTimeout int = 10

var ModelToService map[string][]ModelDetails
var LoadBalancingStrategy string
var ServerPort string
var APIKey string
var Debug bool
var LogLevel string
var SuppertModels map[string]string
var GlobalModelRedirect map[string]string
var SupportMultiContentModels = []string{"gpt-4o", "gpt-4-turbo", "glm-4v", "gemini-*"}

type Limit struct {
	QPS         int `json:"qps"`
	QPM         int `json:"qpm"`
	RPM         int `json:"rpm"`
	Concurrency int `json:"concurrency"`
	Timeout     int `json:"timeout"`
}

// ServiceModel 定义相关结构体
type ServiceModel struct {
	Models             []string            `json:"models"`
	Enabled            bool                `json:"enabled"`
	Credentials        map[string]string   `json:"credentials"`
	CredentialList     []map[string]string `json:"credential_list"`
	ServerURL          string              `json:"server_url"`
	ModelMap           map[string]string   `json:"model_map"`
	ModelRedirect      map[string]string   `json:"model_redirect"`
	Limit              Limit               `json:"limit"`
	Limiter            *rate.Limiter       `json:"-"`
	Timeout            int                 `json:"-"`
	ConcurrencyLimiter chan struct{}       `json:"-"`
}

type Configuration struct {
	ServerPort string `json:"server_port"`
	Debug      bool   `json:"debug"`
	LogLevel   string `json:"log_level"`
	Proxy      struct {
		HTTPProxy  string `json:"http_proxy"`
		HTTPSProxy string `json:"https_proxy"`
	} `json:"proxy"`
	APIKey             string                    `json:"api_key"`
	LoadBalancing      string                    `json:"load_balancing"`
	MultiContentModels []string                  `json:"multi_content_models"`
	ModelRedirect      map[string]string         `json:"model_redirect"`
	Services           map[string][]ServiceModel `json:"services"`
}

// ModelDetails 结构用于返回模型相关的服务信息
type ModelDetails struct {
	ServiceName string
	ServiceModel
}

// 创建模型到服务的映射
func createModelToServiceMap(config Configuration) map[string][]ModelDetails {
	modelToService := make(map[string][]ModelDetails)
	SuppertModels = make(map[string]string)
	for serviceName, serviceModels := range config.Services {
		for _, model := range serviceModels {
			if model.Enabled {
				var limiter *rate.Limiter
				var semaphore chan struct{}
				if model.Limit.QPS > 0 {
					limiter = rate.NewLimiter(rate.Limit(model.Limit.QPS), int(model.Limit.QPS)) // 设定令牌桶的容量等于QPS
				} else if model.Limit.QPM > 0 {
					limiter = rate.NewLimiter(rate.Every(1*time.Minute/time.Duration(model.Limit.QPM)), model.Limit.QPM*2)
				} else if model.Limit.RPM > 0 {
					model.Limit.QPM = model.Limit.RPM
					limiter = rate.NewLimiter(rate.Every(1*time.Minute/time.Duration(model.Limit.RPM)), model.Limit.RPM*2)
				} else {
					if model.Limit.Concurrency > 0 {
						log.Println("create semaphore chan ", model.Limit.Concurrency)
						semaphore = make(chan struct{}, model.Limit.Concurrency)
						log.Println(cap(semaphore))
						for i := 0; i < model.Limit.Concurrency; i++ {
							semaphore <- struct{}{} // 预填充通道，以便其可以被正确地使用
						}
					}
				}

				model.Limiter = limiter
				model.ConcurrencyLimiter = semaphore

				log.Println("Limiter", model.Limiter)
				log.Println("ConcurrencyLimiter", model.ConcurrencyLimiter)

				model.Timeout = defaultLimitTimeout
				if model.Limit.Timeout > 0 {
					model.Timeout = model.Limit.Timeout
				}
				log.Printf("Models: %v, Timeout: %v, QPS: %v, QPM: %v, RPM: %v,Concurrency: %v\n",
					model.Models, model.Timeout, model.Limit.QPS, model.Limit.QPM, model.Limit.RPM, model.Limit.Concurrency)

				if len(model.Models) == 0 {
					dmv, exists := DefaultSupportModelMap[serviceName]
					if exists {
						model.Models = dmv
						log.Println("use default support models:", dmv)
					}
				}

				for _, modelName := range model.Models {
					detail := ModelDetails{
						ServiceName:  serviceName,
						ServiceModel: model,
					}
					modelToService[modelName] = append(modelToService[modelName], detail)

					//存储支持的模型名称列表
					SuppertModels[modelName] = modelName
					for k, v := range detail.ModelRedirect {
						//support models
						SuppertModels[k] = v

						_, exists := SuppertModels[v]
						if exists {
							delete(SuppertModels, v)
						}

						//
						modelToService[k] = append(modelToService[k], detail)
						//delete(modelToService, modelName)
					}
				}
			}
		}
	}
	return modelToService
}

// InitConfig 初始化配置
func InitConfig(configName string) error {
	if configName == "" {
		configName = "config.json"
	}

	//configAbsolutePath, err := utils.GetAbsolutePath(configName)
	configAbsolutePath, err := utils.ResolveRelativePathToAbsolute(configName)
	if err != nil {
		log.Println("Error getting absolute path:", err)
		return err
	}
	log.Println("config name:", configAbsolutePath)
	// 从文件读取配置数据
	data, err := os.ReadFile(configAbsolutePath)
	if err != nil {
		log.Println("Error reading JSON file: ", err)
		return err
	}

	log.Println("read config ok,", configAbsolutePath)

	// 解析 JSON 数据到结构体
	var conf Configuration
	err = json.Unmarshal(data, &conf)
	if err != nil {
		log.Println(err)
	}

	// 设置负载均衡策略，默认为 "first"
	if conf.LoadBalancing == "" {
		LoadBalancingStrategy = "random"
	} else {
		LoadBalancingStrategy = conf.LoadBalancing
	}

	if conf.Proxy.HTTPProxy != "" {
		log.Println("set HTTP_PROXY", conf.Proxy.HTTPProxy)
		os.Setenv("HTTP_PROXY", conf.Proxy.HTTPProxy)

	}
	if conf.Proxy.HTTPSProxy != "" {
		log.Println("set HTTPS_PROXY", conf.Proxy.HTTPSProxy)
		os.Setenv("HTTPS_PROXY", conf.Proxy.HTTPSProxy)
	}

	if conf.APIKey != "" {
		APIKey = conf.APIKey
	}

	log.Println("read LoadBalancingStrategy ok,", LoadBalancingStrategy)

	// 设置服务器端口，默认为 "9090"
	if conf.ServerPort == "" {
		ServerPort = ":9090"
	} else {
		ServerPort = conf.ServerPort
	}
	log.Println("read ServerPort ok,", ServerPort)

	Debug = conf.Debug

	LogLevel = conf.LogLevel
	log.Println("log level: ", LogLevel)

	// 创建映射
	ModelToService = createModelToServiceMap(conf)

	GlobalModelRedirect = conf.ModelRedirect

	log.Println("GlobalModelRedirect: ", GlobalModelRedirect)
	//
	ShowSupportModels()

	if len(conf.MultiContentModels) > 0 {
		SupportMultiContentModels = append(SupportMultiContentModels, conf.MultiContentModels...)
	}
	log.Println("VisionModels: ", SupportMultiContentModels)

	return nil
}

// GetAllModelService 根据模型名称获取服务和凭证信息
func GetAllModelService(modelName string) ([]ModelDetails, error) {
	if serviceDetails, found := ModelToService[modelName]; found {
		return serviceDetails, nil
	}
	return nil, fmt.Errorf("model %s not found in the configuration", modelName)
}

// GetModelService 根据模型名称获取启用的服务和凭证信息
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

	//	log.Println(randomString)

	return md, randomString, nil

}

// GetModelMapping 函数，根据model在ModelMap中查找对应的映射，如果找不到则返回原始model
func GetModelMapping(s *ModelDetails, model string) string {
	if mappedModel, exists := s.ModelMap[model]; exists {
		mylog.Logger.Info("model map found", zap.String("model", model), zap.String("mappedModel", mappedModel))
		return mappedModel
	}
	mylog.Logger.Info("no model map found", zap.String("model", model))
	return model
}

// GetModelMapping 函数，根据model在ModelMap中查找对应的映射，如果找不到则返回原始model
func GetModelRedirect(s *ModelDetails, model string) string {
	if redirectModel, exists := s.ModelRedirect[model]; exists {
		mylog.Logger.Info("ModelRedirect model found", zap.String("model", model), zap.String("redirectModel", redirectModel))
		return redirectModel
	}
	mylog.Logger.Info(" ModelRedirect no model found", zap.String("model", model))
	return model
}

// GetModelMapping 函数，根据model在ModelMap中查找对应的映射，如果找不到则返回原始model
func GetGlobalModelRedirect(model string) string {
	if redirectModel, exists := GlobalModelRedirect[KEYNAME_ALL]; exists {
		if redirectModel == KEYNAME_ALL {
			redirectModel = KEYNAME_RANDOM
		}
		mylog.Logger.Info("GlobalModelRedirect model all found", zap.String("model", model), zap.String("redirectModel", redirectModel))
		return redirectModel
	}

	if redirectModel, exists := GlobalModelRedirect[model]; exists {
		mylog.Logger.Info("GlobalModelRedirect model found", zap.String("model", model), zap.String("redirectModel", redirectModel))
		return redirectModel
	}

	mylog.Logger.Info(" GlobalModelRedirect no model found", zap.String("model", model))
	return model
}

func ShowSupportModels() {
	keys := make([]string, 0, len(ModelToService))

	for k := range SuppertModels {
		keys = append(keys, k)
	}
	sort.Strings(keys) // 对keys进行排序

	log.Println("other support models:", keys)
}

func IsSupportMultiContent(model string) bool {
	for _, item := range SupportMultiContentModels {
		if strings.HasSuffix(item, "*") {
			prefix := strings.TrimSuffix(item, "*")
			if strings.HasPrefix(model, prefix) {
				return true
			}
		} else if item == model {
			return true
		}
	}
	return false
}
