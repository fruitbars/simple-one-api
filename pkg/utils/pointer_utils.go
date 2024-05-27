package utils

// 辅助函数：获取指针值
func GetString(ptr *string) string {
	if ptr != nil {
		return *ptr
	}
	return ""
}

func GetInt64(ptr *int64) int64 {
	if ptr != nil {
		return *ptr
	}
	return 0
}

func GetInt(ptr *int) int {
	if ptr != nil {
		return *ptr
	}
	return 0
}
