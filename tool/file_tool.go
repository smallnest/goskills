package tool

import (
	"fmt"
	"os"
)

// ReadFile reads the content of a file and returns it as a string.
func ReadFile(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file '%s': %w", filePath, err)
	}
	return string(content), nil
}

// WriteFile writes the given content to a file.
// If the file does not exist, it will be created. If it exists, its content will be truncated.
func WriteFile(filePath string, content string) error {
	err := os.WriteFile(filePath, []byte(content), 0644) // 0644 is standard file permissions
	if err != nil {
		return fmt.Errorf("failed to write to file '%s': %w", filePath, err)
	}
	return nil
}
