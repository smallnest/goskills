package tool

import (
	"os"
	"strings"
	"testing"
)

func TestBash(t *testing.T) {
	// Set WORKDIR for testing
	os.Setenv("WORKDIR", ".")
	defer os.Unsetenv("WORKDIR")

	tests := []struct {
		name         string
		command      string
		wantContains string
		wantErr      bool
	}{
		{
			name:         "echo command",
			command:      "echo hello world",
			wantContains: "hello world",
			wantErr:      false,
		},
		{
			name:         "list files",
			command:      "ls",
			wantContains: "",
			wantErr:      false,
		},
		{
			name:         "dangerous command blocked",
			command:      "rm -rf /",
			wantContains: "",
			wantErr:      true,
		},
		{
			name:         "cd command",
			command:      "cd /tmp",
			wantContains: "Changed directory to",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Bash(tt.command)
			if (err != nil) != tt.wantErr {
				t.Errorf("Bash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.wantContains != "" && !strings.Contains(got, tt.wantContains) {
				t.Errorf("Bash() = %v, wantContains %v", got, tt.wantContains)
			}
		})
	}
}

func TestBashFileOperations(t *testing.T) {
	os.Setenv("WORKDIR", ".")
	defer os.Unsetenv("WORKDIR")

	// Test echo (write file via redirection)
	t.Run("echo write", func(t *testing.T) {
		output, err := Bash("echo 'test content' > /tmp/test_bash_echo.txt && cat /tmp/test_bash_echo.txt")
		if err != nil {
			t.Errorf("echo write failed: %v", err)
		}
		if !strings.Contains(output, "test content") {
			t.Errorf("echo write = %v, want 'test content'", output)
		}
		os.Remove("/tmp/test_bash_echo.txt")
	})

	// Test cat (read file)
	t.Run("cat read", func(t *testing.T) {
		_ = os.WriteFile("/tmp/test_bash_cat.txt", []byte("file content"), 0644)
		output, err := Bash("cat /tmp/test_bash_cat.txt")
		if err != nil {
			t.Errorf("cat read failed: %v", err)
		}
		if !strings.Contains(output, "file content") {
			t.Errorf("cat read = %v, want 'file content'", output)
		}
		os.Remove("/tmp/test_bash_cat.txt")
	})

	// Test grep
	t.Run("grep search", func(t *testing.T) {
		_ = os.WriteFile("/tmp/test_bash_grep.txt", []byte("line1\nline2\nline3\n"), 0644)
		output, err := Bash("grep 'line2' /tmp/test_bash_grep.txt")
		if err != nil {
			t.Errorf("grep search failed: %v", err)
		}
		if !strings.Contains(output, "line2") {
			t.Errorf("grep search = %v, want 'line2'", output)
		}
		os.Remove("/tmp/test_bash_grep.txt")
	})
}

func TestBashScriptExecution(t *testing.T) {
	os.Setenv("WORKDIR", ".")
	defer os.Unsetenv("WORKDIR")

	// Test bash script execution
	t.Run("bash script", func(t *testing.T) {
		script := `#!/bin/bash
echo "script output"`
		_ = os.WriteFile("/tmp/test_script.sh", []byte(script), 0755)
		output, err := Bash("bash /tmp/test_script.sh")
		if err != nil {
			t.Errorf("bash script failed: %v", err)
		}
		if !strings.Contains(output, "script output") {
			t.Errorf("bash script = %v, want 'script output'", output)
		}
		os.Remove("/tmp/test_script.sh")
	})
}
