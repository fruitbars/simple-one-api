package utils

import "time"

// parseToUnixTime函数接收一个符合RFC3339Nano格式的日期时间字符串，并返回其对应的Unix时间戳（int类型）。
func ParseRFC3339NanoToUnixTime(dateTimeStr string) (int64, error) {
	// 使用time.Parse解析符合RFC3339Nano格式的时间字符串
	t, err := time.Parse(time.RFC3339Nano, dateTimeStr)
	if err != nil {
		return 0, err // 如果解析错误，返回错误信息
	}
	return t.Unix(), nil // 返回Unix时间戳
}
