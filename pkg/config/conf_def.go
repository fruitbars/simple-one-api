package config

var ServiceTimeOut int = 30

var PROXY_STRATEGY_FORCEALL = "force_all"
var PROXY_STRATEGY_ALL = "all"
var PROXY_STRATEGY_DEFAULT = "default"
var PROXY_STRATEGY_DISABLED = "disabled"

var SupportedServiceTypes = map[string]struct{}{
	"azure": {}, "bailian": {}, "claude": {},
	"dashscope": {}, "deepseek": {}, "dify": {}, "gemini": {}, "groq": {}, "hunyuan": {}, "huoshan": {},
	"minimax": {}, "ollama": {}, "openai": {}, "qianfan": {}, "vertexai": {}, "xinghuo": {}, "zhipu": {},
}
