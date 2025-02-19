package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"simple-one-api/pkg/mylog"
	"simple-one-api/pkg/utils"
	"sort"
	"strings"
)

var GSOAConf *Configuration

var ModelToService map[string][]ModelDetails
var LoadBalancingStrategy string
var ServerPort string
var APIKey string
var Debug bool
var LogLevel string
var SupportModels map[string]string
var GlobalModelRedirect map[string]string
var SupportMultiContentModels = []string{"gpt-4o", "gpt-4-turbo", "glm-4v", "gemini-*", "yi-vision", "gpt-4o*"}
var GProxyConf *ProxyConf
var GTranslation *Translation

var apiKeyMap map[string]APIKeyConfig

type Limit struct {
	QPS         float64 `json:"qps" yaml:"qps"`
	QPM         float64 `json:"qpm" yaml:"qpm"`
	RPM         float64 `json:"rpm" yaml:"rpm"`
	Concurrency float64 `json:"concurrency" yaml:"concurrency"`
	Timeout     int     `json:"timeout" yaml:"timeout"`
}

type Range struct {
	Min float64 `json:"min" yaml:"min"`
	Max float64 `json:"max" yaml:"max"`
}

type ModelParams struct {
	TemperatureRange Range `json:"temperatureRange" yaml:"temperatureRange"`
	TopPRange        Range `json:"topPRange" yaml:"topPRange"`
	MaxTokens        int   `json:"maxTokens" yaml:"maxTokens"`
}

// ServiceModel 定义相关结构体
type ServiceModel struct {
	Provider        string                   `json:"provider" yaml:"provider"`
	EmbeddingModels []string                 `json:"embedding_models" yaml:"embedding_models"`
	EmbeddingLimit  Limit                    `json:"embedding_limit" yaml:"embedding_limit"`
	Models          []string                 `json:"models" yaml:"models"`
	Enabled         bool                     `json:"enabled" yaml:"enabled"`
	Credentials     map[string]interface{}   `json:"credentials" yaml:"credentials"`
	CredentialList  []map[string]interface{} `json:"credential_list" yaml:"credential_list"`
	ServerURL       string                   `json:"server_url" yaml:"server_url"`
	ModelMap        map[string]string        `json:"model_map" yaml:"model_map"`
	ModelRedirect   map[string]string        `json:"model_redirect" yaml:"model_redirect"`
	Limit           Limit                    `json:"limit" yaml:"limit"`
	UseProxy        *bool                    `json:"use_proxy,omitempty" yaml:"use_proxy,omitempty"`
	Timeout         int                      `json:"timeout" yaml:"timeout"`
}

type ProxyConf struct {
	Strategy    string `json:"strategy" yaml:"strategy"`
	Type        string `json:"type" yaml:"type"`
	HTTPProxy   string `json:"http_proxy" yaml:"http_proxy"`
	HTTPSProxy  string `json:"https_proxy" yaml:"https_proxy"`
	Socks5Proxy string `json:"socks5_proxy" yaml:"socks5_proxy"`
	Timeout     int    `json:"timeout" yaml:"timeout"`
}

type Translation struct {
	Enable         bool   `json:"enable" yaml:"enable"`
	PromptTemplate string `json:"promptTemplate" yaml:"prompt_template"`
	Retry          int    `json:"retry" yaml:"retry"`
	Concurrency    int    `json:"concurrency" yaml:"concurrency"`
}

type APIKeyConfig struct {
	APIKey          string              `json:"api_key" yaml:"api_key"`
	SupportedModels map[string][]string `json:"supported_models" yaml:"supported_models"`
}

type Configuration struct {
	ServerPort         string                    `json:"server_port" yaml:"server_port"`
	Debug              bool                      `json:"debug" yaml:"debug"`
	LogLevel           string                    `json:"log_level" yaml:"log_level"`
	Proxy              ProxyConf                 `json:"proxy" yaml:"proxy"`
	APIKey             string                    `json:"api_key" yaml:"api_key"`
	LoadBalancing      string                    `json:"load_balancing" yaml:"load_balancing"`
	MultiContentModels []string                  `json:"multi_content_models" yaml:"multi_content_models"`
	ModelRedirect      map[string]string         `json:"model_redirect" yaml:"model_redirect"`
	ParamsRange        map[string]ModelParams    `json:"params_range" yaml:"params_range"`
	Services           map[string][]ServiceModel `json:"services" yaml:"services"`
	Translation        Translation               `json:"translation" yaml:"translation"`
	EnableWeb          bool                      `json:"enable_web" yaml:"enable_web"`
	APIKeys            []APIKeyConfig            `json:"api_keys" yaml:"api_keys"`
}

// ModelDetails 结构用于返回模型相关的服务信息
type ModelDetails struct {
	ServiceName  string `json:"service_name" yaml:"service_name"`
	ServiceModel `json:",inline" yaml:",inline"`
	ServiceID    string `json:"service_id" yaml:"service_id"`
}

// 创建模型到服务的映射
func createModelToServiceMap(config Configuration) map[string][]ModelDetails {
	modelToService := make(map[string][]ModelDetails)
	SupportModels = make(map[string]string)
	for serviceName, serviceModels := range config.Services {
		for _, model := range serviceModels {
			if model.Enabled {
				log.Printf("Models: %v, service Timeout:%v,Limit Timeout: %v, QPS: %v, QPM: %v, RPM: %v,Concurrency: %v\n",
					model.Models, model.Timeout, model.Limit.Timeout, model.Limit.QPS, model.Limit.QPM, model.Limit.RPM, model.Limit.Concurrency)

				log.Printf("Models: %v\n", model.EmbeddingModels)

				if len(model.Models) == 0 {
					dmv, exists := DefaultSupportModelMap[serviceName]
					if exists {
						model.Models = dmv
						log.Println("use default support models:", dmv)
					}
				}

				if model.Timeout <= 0 {
					model.Timeout = ServiceTimeOut
				}

				for _, modelName := range model.Models {
					detail := ModelDetails{
						ServiceName:  serviceName,
						ServiceModel: model,
						ServiceID:    uuid.New().String(),
					}

					//modelNameLower := strings.ToLower(modelName)
					modelToService[modelName] = append(modelToService[modelName], detail)

					//存储支持的模型名称列表
					SupportModels[modelName] = modelName
					for k, v := range detail.ModelRedirect {
						//support models
						SupportModels[k] = v

						_, exists := SupportModels[v]
						if exists {
							delete(SupportModels, v)
						}

						//
						modelToService[k] = append(modelToService[k], detail)
						//delete(modelToService, modelName)
					}
				}

				for _, modelName := range model.EmbeddingModels {
					detail := ModelDetails{
						ServiceName:  serviceName,
						ServiceModel: model,
						ServiceID:    uuid.New().String(),
					}

					//modelNameLower := strings.ToLower(modelName)
					modelToService[modelName] = append(modelToService[modelName], detail)
					for k, _ := range detail.ModelRedirect {
						modelToService[k] = append(modelToService[k], detail)
					}
				}
			}
		}
	}
	return modelToService
}

// InitConfig 初始化配置
func InitConfig(configName string) error {

	// 解析 JSON 数据到结构体
	var conf Configuration

	configAbsolutePath, err := utils.ResolveRelativePathToAbsolute(configName)
	if err != nil {
		log.Println("Error getting absolute path:", err)
		return err
	}

	if !utils.FileExists(configAbsolutePath) {
		log.Println("config name:", configAbsolutePath, "not exist")
		configName = "config/" + configName
		configAbsolutePath, err = utils.ResolveRelativePathToAbsolute(configName)
		if err != nil {
			log.Println("Error getting absolute path:", err)
			return err
		}
	}

	log.Println("config name:", configAbsolutePath)
	// 从文件读取配置数据
	data, err := os.ReadFile(configAbsolutePath)
	if err != nil {
		log.Println("Error reading JSON file: ", err)
		return err
	}

	fname, ftype := utils.GetFileNameAndType(configName)
	log.Println(fname, ftype)

	if ftype == "yml" || ftype == "yaml" {

		err = yaml.Unmarshal(data, &conf)
		if err != nil {
			log.Println("Unable to decode into struct:", err)
			return err
		}

	} else if ftype == "json" {
		err = json.Unmarshal(data, &conf)
		if err != nil {
			log.Println(err)

			if syntaxErr, ok := err.(*json.SyntaxError); ok {
				line, character := FindLineAndCharacter(data, int(syntaxErr.Offset))
				log.Printf("JSON 语法错误在第 %d 行，第 %d 个字符附近: %v\n", line, character, err)
				log.Printf("上下文: %s\n", GetErrorContext(data, int(syntaxErr.Offset)))
			} else {
				log.Printf("JSON 解析错误: %v\n", err)
			}
		}
	} else {
		log.Println("unsupport config type:", ftype)
		return errors.New("unsupport config type")
	}

	log.Println(conf)

	// 设置负载均衡策略，默认为 "first"
	if conf.LoadBalancing == "" {
		LoadBalancingStrategy = "random"
	} else {
		LoadBalancingStrategy = conf.LoadBalancing
	}

	GSOAConf = &conf

	GProxyConf = &(conf.Proxy)

	log.Println(conf.Proxy)

	if conf.APIKey != "" {
		APIKey = conf.APIKey
	}

	initAPIKeyMap()

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

	GTranslation = &conf.Translation

	log.Println("GlobalModelRedirect: ", GlobalModelRedirect)
	//
	ShowSupportModels()

	if len(conf.MultiContentModels) > 0 {
		SupportMultiContentModels = append(SupportMultiContentModels, conf.MultiContentModels...)
	}
	log.Println("SupportMultiContentModels: ", SupportMultiContentModels)

	return nil
}

/*
// GetAllModelService 根据模型名称获取服务和凭证信息
func GetAllModelService(modelName string) ([]ModelDetails, error) {
	if serviceDetails, found := ModelToService[modelName]; found {
		return serviceDetails, nil
	}
	return nil, fmt.Errorf("model %s not found in the configuration", modelName)
}

*/

// GetModelService 根据模型名称获取启用的服务和凭证信息
func GetModelService(modelName string) (*ModelDetails, error) {
	if serviceDetails, found := ModelToService[modelName]; found {
		var enabledServices []ModelDetails
		for _, sd := range serviceDetails {
			if sd.Enabled {
				enabledServices = append(enabledServices, sd)
			}
		}

		if len(enabledServices) == 0 {
			return nil, fmt.Errorf("no enabled model %s found in the configuration", modelName)
		}

		index := GetLBIndex(LoadBalancingStrategy, modelName, len(enabledServices))

		return &enabledServices[index], nil
	}
	return nil, fmt.Errorf("model %s not found in the configuration", modelName)
}

func GetRandomEnabledModelDetails() (*ModelDetails, error) {

	index := GetLBIndex(LoadBalancingStrategy, KEYNAME_RANDOM, len(ModelToService))

	keys := make([]string, 0, len(ModelToService))

	// 遍历 ModelToService 映射，收集所有 Enabled 为 true 的 ModelDetails
	for modelName := range ModelToService {
		keys = append(keys, modelName)
	}

	sort.Strings(keys)

	model := keys[index]

	modelDetails := ModelToService[model]

	index2 := GetLBIndex(LoadBalancingStrategy, model, len(modelDetails))

	randomModel := modelDetails[index2]

	return &randomModel, nil
}

func GetRandomEnabledModelDetailsV1() (*ModelDetails, string, error) {
	md, err := GetRandomEnabledModelDetails()
	if err != nil {
		return nil, "", err
	}

	randomString := md.Models[getRandomIndex(len(md.Models))]

	//	log.Println(randomString)

	return md, randomString, nil

}

// GetModelMapping 函数，根据model在ModelMap中查找对应的映射，如果找不到则返回原始model
func GetModelMapping(s *ModelDetails, model string) string {
	if mappedModel, exists := s.ModelMap[model]; exists {
		mylog.Logger.Info("model map found", zap.String("model", model), zap.String("mappedModel", mappedModel))
		return mappedModel
	}
	mylog.Logger.Debug("no model map found", zap.String("model", model))
	return model
}

// GetModelRedirect 函数，根据model在ModelMap中查找对应的映射，如果找不到则返回原始model
func GetModelRedirect(s *ModelDetails, model string) string {
	if redirectModel, exists := s.ModelRedirect[model]; exists {
		mylog.Logger.Info("ModelRedirect model found", zap.String("model", model), zap.String("redirectModel", redirectModel))
		return redirectModel
	}
	mylog.Logger.Debug(" ModelRedirect no model found", zap.String("model", model))
	return model
}

// GetGlobalModelRedirect 函数，根据model在ModelMap中查找对应的映射，如果找不到则返回原始model
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

	mylog.Logger.Debug(" GlobalModelRedirect no model found", zap.String("model", model))
	return model
}

func ShowSupportModels() {
	keys := make([]string, 0, len(ModelToService))

	for k := range SupportModels {
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

func IsProxyEnabled(s *ModelDetails) bool {
	switch GProxyConf.Strategy {
	case PROXY_STRATEGY_FORCEALL:
		// 配置全部启用代理，即使服务内配置了false，也忽略
		return true
	case PROXY_STRATEGY_ALL:
		// 配置全部启用代理，如果服务内配置了false，则不启动，其他情况全部启用
		if s.UseProxy == nil || (s.UseProxy != nil && *s.UseProxy) {
			return true
		}
	case PROXY_STRATEGY_DEFAULT:
		// 配置根据配置启用代理，默认是关闭
		if s.UseProxy != nil && *s.UseProxy {
			return true
		}
	case PROXY_STRATEGY_DISABLED:
		// 配置全部禁用代理
		return false
	default:
		return false
	}

	return false
}

func initAPIKeyMap() {
	apiKeyMap = make(map[string]APIKeyConfig)
	for _, keyConfig := range GSOAConf.APIKeys {
		apiKeyMap[keyConfig.APIKey] = keyConfig
	}
}

func ValidateAPIKeyAndModel(apikey string, model string) (bool, string) {
	if len(apiKeyMap) == 0 {
		return true, ""
	}
	keyConfig, exists := apiKeyMap[apikey]
	if !exists {
		mylog.Logger.Error("ValidateAPIKeyAndModel|Forbidden: invalid API key", zap.String("apikey", apikey))
		return false, "Forbidden: invalid API key"
	}

	mylog.Logger.Debug("ValidateAPIKeyAndModel", zap.String("model", model))

	// 检查所有服务和通配符的配置
	for service, models := range keyConfig.SupportedModels {
		mylog.Logger.Info(service, zap.Any("SupportedModels", models))
		for _, m := range models {
			if m == "*" || m == model {
				mylog.Logger.Debug("ValidateAPIKeyAndModel", zap.String("model", model), zap.String("m", m))
				return true, ""
			}
		}
	}
	return false, "Forbidden: model not supported"
}
