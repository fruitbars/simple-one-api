package baidu_qianfan

import "errors"

// 定义服务对应关系
var serviceMap = map[string]string{
	"ERNIE-4.0-8K":                   "completions_pro",
	"ERNIE-4.0-8K-Latest":            "ernie-4.0-8k-latest",
	"ERNIE-4.0-8K-Preview":           "ernie-4.0-8k-preview",
	"ERNIE-4.0-8K-0329":              "ernie-4.0-8k-0329",
	"ERNIE-4.0-8K-0613":              "ernie-4.0-8k-0613",
	"ERNIE-4.0-Turbo-8K":             "ernie-4.0-turbo-8k",
	"ERNIE-4.0-Turbo-8K-Preview":     "ernie-4.0-turbo-8k-preview",
	"ERNIE-3.5-8K":                   "completions",
	"ERNIE-3.5-8K-Preview":           "ernie-3.5-8k-preview",
	"ERNIE-3.5-8K-0329":              "ernie-3.5-8k-0329",
	"ERNIE-3.5-128K":                 "ernie-3.5-128k",
	"ERNIE-3.5-8K-0613":              "ernie-3.5-8k-0613",
	"ERNIE-3.5-8K-0701":              "ernie-3.5-8k-0701",
	"ERNIE-Speed-8K":                 "ernie_speed",
	"ERNIE-Speed-128K":               "ernie-speed-128k",
	"ERNIE-Lite-8K-0922":             "eb-instant",
	"ERNIE-Lite-8K":                  "ernie-lite-8k",
	"ERNIE-Lite-8K-0725":             "{}",
	"ERNIE-Lite-4K-0704":             "{}",
	"ERNIE-Lite-4K-0516":             "{}",
	"ERNIE-Lite-128K-0419":           "{}",
	"ERNIE-Tiny-8K":                  "ernie-tiny-8k",
	"ERNIE-Novel-8K":                 "ernie-novel-8k",
	"ERNIE-Character-8K":             "ernie-char-8k",
	"ERNIE-Functions-8K":             "ernie-func-8k",
	"Qianfan-Dynamic-8K":             "qianfan-dynamic-8k",
	"ERNIE-Speed-AppBuilder-8K":      "ai_apaas",
	"ERNIE-Lite-AppBuilder-8K-0614":  "ai_apaas_lite",
	"Gemma-2B-it":                    "{}",
	"Gemma-7B-it":                    "gemma_7b_it",
	"Yi-34B-Chat":                    "yi_34b_chat",
	"Mixtral-8x7B-Instruct":          "mixtral_8x7b_instruct",
	"Mistral-7B-Instruct":            "{}",
	"Llama-2-7b-chat":                "llama_2_7b",
	"Linly-Chinese-LLaMA-2-7B":       "{}",
	"Qianfan-Chinese-Llama-2-7B":     "qianfan_chinese_llama_2_7b",
	"Qianfan-Chinese-Llama-2-7B-32K": "{}",
	"Llama-2-13b-chat":               "llama_2_13b",
	"Linly-Chinese-LLaMA-2-13B":      "{}",
	"Qianfan-Chinese-Llama-2-13B-v1": "qianfan_chinese_llama_2_13b",
	"Qianfan-Chinese-Llama-2-13B-v2": "{}",
	"Llama-2-70b-chat":               "llama_2_70b",
	"Qianfan-Llama-2-70B-compressed": "{}",
	"Qianfan-Chinese-Llama-2-70B":    "qianfan_chinese_llama_2_70b",
	"Qianfan-Chinese-Llama-2-1.3B":   "{}",
	"Meta-Llama-3-8B-Instruct":       "llama_3_8b",
	"Meta-Llama-3-70B-Instruct":      "llama_3_70b",
	"ChatGLM3-6B":                    "{}",
	"chatglm3-6b-32k":                "{}",
	"ChatGLM2-6B-32K":                "chatglm2_6b_32k",
	"ChatGLM2-6B-INT4":               "{}",
	"ChatGLM2-6B":                    "{}",
	"Baichuan2-7B-Chat":              "{}",
	"Baichuan2-13B-Chat":             "{}",
	"XVERSE-13B-Chat":                "{}",
	"XuanYuan-70B-Chat-4bit":         "xuanyuan_70b_chat",
	"DISC-MedLLM":                    "{}",
	"ChatLaw":                        "chatlaw",
	"Falcon-7B":                      "{}",
	"Falcon-40B-Instruct":            "{}",
	"AquilaChat-7B":                  "aquilachat_7b",
	"RWKV-4-World":                   "{}",
	"BLOOMZ-7B":                      "bloomz_7b1",
	"Qianfan-BLOOMZ-7B-compressed":   "qianfan_bloomz_7b_compressed",
	"RWKV-4-pile-14B":                "{}",
	"RWKV-Raven-14B":                 "{}",
	"OpenLLaMA-7B":                   "{}",
	"Dolly-12B":                      "{}",
	"MPT-7B-Instruct":                "{}",
	"MPT-30B-instruct":               "{}",
	"OA-Pythia-12B-SFT-4":            "{}",
}

// 根据输入的服务名返回对应的字符串
func qianfanModel2Address(serviceName string) (string, error) {
	// 查找并返回映射值
	if code, exists := serviceMap[serviceName]; exists {
		return code, nil
	}

	return "", errors.New("服务名未找到")
}
