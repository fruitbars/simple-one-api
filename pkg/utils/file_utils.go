package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func GetAbsolutePathDir(filename string) (string, error) {
	// 如果文件名是绝对路径，直接返回其目录名
	if filepath.IsAbs(filename) {
		return filepath.Dir(filename), nil
	}

	// 获取当前工作目录
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("could not get current working directory: %w", err)
	}

	// 将相对路径转换为绝对路径
	absPath := filepath.Join(wd, filename)

	// 返回绝对路径的目录名
	return filepath.Dir(absPath), nil
}

func GetFileNameAndType(filePath string) (string, string) {
	// 使用filepath.Base获取文件名（包含后缀）
	baseName := filepath.Base(filePath)

	// 使用filepath.Ext获取文件的后缀名
	fileType := filepath.Ext(baseName)

	// 去掉后缀名中的点
	fileType = strings.TrimPrefix(fileType, ".")

	// 获取文件名（不包含后缀）
	fileName := strings.TrimSuffix(baseName, fileType)
	fileName = strings.TrimSuffix(fileName, ".")

	return fileName, fileType
}

// IsSimpleFileName checks if the given file name is just a simple file name without any directory path.
func IsSimpleFileName(fileName string) bool {
	// Check if the file name is an absolute path
	if strings.HasPrefix(fileName, "/") {
		return false
	}

	// Check if the file name contains any directory separators
	if strings.Contains(fileName, "/") {
		return false
	}

	return true
}

func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return true
}
