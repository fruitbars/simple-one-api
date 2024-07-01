package utils

// GetStringFromMap 试图从给定的 map 中提取指定键的字符串值。
func GetStringFromMap(data map[string]interface{}, key string) (string, bool) {
	if value, exists := data[key]; exists {
		if strValue, ok := value.(string); ok {
			return strValue, true
		}
		return "", false
	}
	return "", false
}
