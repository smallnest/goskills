package tool

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Bash executes a shell command and returns its combined stdout and stderr.
// This is the universal tool for:
// - File operations (cat, grep, echo, find, etc.)
// - Script execution (python3, node, npx tsx, bash)
// - Git operations
// - Package management (npm, pip, etc.)
func Bash(command string) (string, error) {
	// Safety check for dangerous commands
	dangerous := []string{"rm -rf /", "rm -rf /*", "> /dev/sd", "> /dev/null", "mkfs", "dd if="}
	for _, d := range dangerous {
		if strings.Contains(command, d) {
			return "", fmt.Errorf("dangerous command blocked: %s", d)
		}
	}

	// Determine working directory
	workDir := os.Getenv("WORKDIR")
	if workDir == "" {
		workDir = "."
	}

	// Parse command into parts
	var cmd *exec.Cmd
	if strings.HasPrefix(command, "cd ") {
		// Handle cd specially by updating workDir for subsequent commands
		parts := strings.Fields(command)
		if len(parts) >= 2 {
			newDir := strings.Join(parts[1:], " ")
			if filepath.IsAbs(newDir) {
				workDir = newDir
			} else {
				workDir = filepath.Join(workDir, newDir)
			}
			os.Setenv("WORKDIR", workDir)
			return fmt.Sprintf("Changed directory to: %s", workDir), nil
		}
	}

	// Execute the command
	cmd = exec.Command("bash", "-c", command)
	cmd.Dir = workDir
	cmd.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Set timeout to prevent hanging
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case <-time.After(2 * time.Minute):
		_ = cmd.Process.Kill() // Best effort cleanup
		return "", fmt.Errorf("command timed out (2 minutes)")
	case err := <-done:
		output := stdout.String() + stderr.String()
		if err != nil {
			return "", fmt.Errorf("command failed: %w\nOutput: %s", err, output)
		}
		return output, nil
	}
}

// BashWithTimeout executes a shell command with a custom timeout.
func BashWithTimeout(command string, timeout time.Duration) (string, error) {
	// Safety check
	dangerous := []string{"rm -rf /", "rm -rf /*", "> /dev/sd", "> /dev/null", "mkfs", "dd if="}
	for _, d := range dangerous {
		if strings.Contains(command, d) {
			return "", fmt.Errorf("dangerous command blocked: %s", d)
		}
	}

	workDir := os.Getenv("WORKDIR")
	if workDir == "" {
		workDir = "."
	}

	cmd := exec.Command("bash", "-c", command)
	cmd.Dir = workDir
	cmd.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case <-time.After(timeout):
		_ = cmd.Process.Kill() // Best effort cleanup
		return "", fmt.Errorf("command timed out (%v)", timeout)
	case err := <-done:
		output := stdout.String() + stderr.String()
		if err != nil {
			return "", fmt.Errorf("command failed: %w\nOutput: %s", err, output)
		}
		return output, nil
	}
}
