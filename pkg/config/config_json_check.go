package config

import (
	"bytes"
	"strings"
)

func FindLineAndCharacter(data []byte, offset int) (int, int) {
	lines := bytes.Split(data, []byte{'\n'})
	lineNumber := 1
	characterPosition := offset

	for _, line := range lines {
		if len(line)+1 < characterPosition { // +1 是因为换行符
			lineNumber++
			characterPosition -= len(line) + 1
		} else {
			break
		}
	}

	return lineNumber, characterPosition
}

// getErrorContext 获取错误上下文的文本
func GetErrorContext(data []byte, offset int) string {
	start := offset - 20 // 显示错误位置前后的文本
	end := offset + 20

	if start < 0 {
		start = 0
	}
	if end > len(data) {
		end = len(data)
	}

	return strings.TrimSpace(string(data[start:end]))
}
