package util

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/pkg/exception"
)

func SafePath(workingDirectory string, filePath string) (string, error) {
	absolutePath, err := filepath.Abs(workingDirectory)
	if err != nil {
		return "", exception.InvalidFilePath
	}

	targetPath := filepath.Join(absolutePath, filePath)
	rel, err := filepath.Rel(absolutePath, targetPath)

	if err != nil || strings.HasPrefix(rel, "..") || rel == ".." {
		return "", exception.InvalidFilePath
	}

	return rel, nil
}

func FormatFileSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d bytes", size)
	}

	kb := float64(size) / 1024
	if kb < 1024 {
		s := fmt.Sprintf("%.1f", kb)
		s = strings.TrimSuffix(s, ".0")
		return s + "KB"
	}

	mb := kb / 1024
	if mb < 1024 {
		s := fmt.Sprintf("%.1f", mb)
		s = strings.TrimSuffix(s, ".0")
		return s + "MB"
	}

	gb := mb / 1024
	s := fmt.Sprintf("%.1f", gb)
	s = strings.TrimSuffix(s, ".0")
	return s + "GB"
}
