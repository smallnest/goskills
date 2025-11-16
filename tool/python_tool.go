package tool

import (
	"bytes"
	"fmt"
	"os/exec"
)

// RunPythonScript executes a Python script and returns its combined stdout and stderr.
// It tries to use 'python3' first, then falls back to 'python'.
func RunPythonScript(scriptPath string, args []string) (string, error) {
	var cmd *exec.Cmd

	// Try python3 first
	cmd = exec.Command("python3", append([]string{scriptPath}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// If python3 fails, try python
		cmd = exec.Command("python", append([]string{scriptPath}, args...)...)
		stdout.Reset()
		stderr.Reset()
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err = cmd.Run()
		if err != nil {
			return "", fmt.Errorf("failed to run python script '%s' with both 'python3' and 'python': %w\nStdout: %s\nStderr: %s", scriptPath, err, stdout.String(), stderr.String())
		}
	}

	return stdout.String() + stderr.String(), nil
}

