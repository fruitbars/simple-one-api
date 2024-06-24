package config

var DefaultSupportModelMap = map[string][]string{
	"qianfan":  {"yi_34b_chat", "ERNIE-Speed-8K", "ERNIE-Speed-128K", "ERNIE-Lite-8K", "ERNIE-Lite-8K-0922", "ERNIE-Tiny-8K"},
	"hunyuan":  {"hunyuan-lite", "hunyuan-standard", "hunyuan-standard-256K", "hunyuan-pro"},
	"xinghuo":  {"spark-lite", "spark-v2.0", "spark-pro", "spark-max"},
	"deepseek": {"deepseek-chat", "deepseek-coder"},
	"zhipu":    {"glm-3-turbo", "glm-4-0520", "glm-4", "glm-4-air", "glm-4-airx", "glm-4-flash", "glm-4v"},
	"minimax":  {"abab6.5", "abab6.5s", "abab6.5t", "abab6.5g", "abab5.5s"},
	"huoshan":  {"Doubao-pro-4k", "Doubao-pro-32k", "Doubao-pro-128k", "Doubao-lite-4k", "Doubao-lite-32k", "Doubao-lite-128k"},
	"gemini":   {"gemini-1.5-pro", "gemini-1.5-flash", "gemini-1.0-pro", "gemini-pro-vision"},
	"groq":     {"llama3-70b-8192", "llama3-8b-8192", "gemma-7b-it", "mixtral-8x7b-32768"},
	"aliyun":   {"qwen-turbo", "qwen-plus", "qwen-max", "qwen-max-longcontext"},
}
