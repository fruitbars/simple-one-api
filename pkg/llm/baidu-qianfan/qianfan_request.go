package baidu_qianfan

import (
	"simple-one-api/pkg/mycommon"
)

// Request 定义了API请求的主体结构
type QianFanRequest struct {
	Messages        []mycommon.Message `json:"messages"`                    // 对话消息列表
	Stream          *bool              `json:"stream,omitempty"`            // 是否以流式接口返回数据
	Temperature     *float64           `json:"temperature,omitempty"`       // 输出随机性控制
	TopP            *float64           `json:"top_p,omitempty"`             // 输出多样性控制
	PenaltyScore    *float64           `json:"penalty_score,omitempty"`     // 减少重复的惩罚分数
	System          *string            `json:"system,omitempty"`            // 系统人设设置
	Stop            []string           `json:"stop,omitempty"`              // 生成停止标识
	MaxOutputTokens *int               `json:"max_output_tokens,omitempty"` // 最大输出token数
	UserID          *string            `json:"user_id,omitempty"`           // 用户唯一标识符
}
