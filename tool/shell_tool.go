package tool

import (
	"bytes"
	"fmt"
	"os/exec"
)

// RunShellScript executes a shell script and returns its combined stdout and stderr.
func RunShellScript(scriptPath string, args []string) (string, error) {
	cmd := exec.Command("bash", append([]string{scriptPath}, args...)...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to run shell script '%s': %w\nStdout: %s\nStderr: %s", scriptPath, err, stdout.String(), stderr.String())
	}

	return stdout.String() + stderr.String(), nil
}
