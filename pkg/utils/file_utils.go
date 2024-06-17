package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

func GetAbsolutePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return absPath, nil
}

func ResolveRelativePathToAbsolute(filename string) (string, error) {
	// 如果文件名是绝对路径，直接返回
	if filepath.IsAbs(filename) {
		return filename, nil
	}

	// 获取当前工作目录
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("could not get current working directory: %w", err)
	}

	// 将相对路径转换为绝对路径
	absPath := filepath.Join(wd, filename)

	return absPath, nil
}
