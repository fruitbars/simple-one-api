package mycommon

import (
	"errors"
	"go.uber.org/zap"
	"simple-one-api/pkg/mylog"
)

const adjustmentFloatValue = 0.01 // 定义调整值为常量

type ModelParams struct {
	TemperatureRange Range // 温度参数范围
	TopPRange        Range // TopP 参数范围
	MaxTokens        int   // 最大 tokens 数量
}

type Range struct {
	Min float32
	Max float32
}

// 共享的模型参数配置
var glmCommonModelParams = ModelParams{
	TemperatureRange: Range{0.0, 1.0},
	TopPRange:        Range{0.0, 1.0},
}

var modelParamsMap = map[string]ModelParams{
	"glm-4-0520": {
		TemperatureRange: glmCommonModelParams.TemperatureRange,
		TopPRange:        glmCommonModelParams.TopPRange,
		MaxTokens:        4095,
	},
	"glm-4": {
		TemperatureRange: glmCommonModelParams.TemperatureRange,
		TopPRange:        glmCommonModelParams.TopPRange,
		MaxTokens:        4095,
	},
	"glm-4-air": {
		TemperatureRange: glmCommonModelParams.TemperatureRange,
		TopPRange:        glmCommonModelParams.TopPRange,
		MaxTokens:        4095,
	},
	"glm-4-airx": {
		TemperatureRange: glmCommonModelParams.TemperatureRange,
		TopPRange:        glmCommonModelParams.TopPRange,
		MaxTokens:        4095,
	},
	"glm-4-flash": {
		TemperatureRange: glmCommonModelParams.TemperatureRange,
		TopPRange:        glmCommonModelParams.TopPRange,
		MaxTokens:        4095,
	},
	"glm-3-turbo": {
		TemperatureRange: glmCommonModelParams.TemperatureRange,
		TopPRange:        glmCommonModelParams.TopPRange,
		MaxTokens:        4095,
	},
	"glm-4v": {
		TemperatureRange: Range{0.0, 1.0},
		TopPRange:        Range{0.0, 1.0},
		MaxTokens:        1024,
	},
}

func GetModelParams(modelName string) (ModelParams, error) {
	params, ok := modelParamsMap[modelName]
	if !ok {
		return ModelParams{}, errors.New("unsupported model")
	}
	return params, nil
}

func adjustFloatValue(value, min, max float32) float32 {
	if value < 0 {
		value = 0
	}
	if value < min {
		value = min + adjustmentFloatValue
	} else if value >= max {
		value = max - adjustmentFloatValue
	}
	return value
}

func AdjustParamsToRange(modelName string, temperature, topP float32, maxTokens int) (float32, float32, int, error) {
	params, err := GetModelParams(modelName)
	if err != nil {
		return temperature, topP, maxTokens, err
	}

	temperature = adjustFloatValue(temperature, params.TemperatureRange.Min, params.TemperatureRange.Max)

	topP = adjustFloatValue(topP, params.TopPRange.Min, params.TopPRange.Max)

	if maxTokens < 0 {
		maxTokens = 0
	}
	if maxTokens > params.MaxTokens {
		maxTokens = params.MaxTokens
	}

	mylog.Logger.Debug("", zap.Float32("adjusted_temperature", temperature),
		zap.Float32("adjusted_topP", topP),
		zap.Int("adjusted_maxTokens", maxTokens))

	return temperature, topP, maxTokens, nil
}
