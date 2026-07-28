package compose_runner

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type Result struct {
	Output string `json:"output"`
}

func Available(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "docker", "compose", "version")
	return cmd.Run() == nil
}

func Config(ctx context.Context, projectDir string, files []string, timeout time.Duration) (Result, error) {
	return run(ctx, projectDir, files, []string{"config"}, timeout)
}

func Up(ctx context.Context, projectDir string, files []string, timeout time.Duration) (Result, error) {
	return run(ctx, projectDir, files, []string{"up", "-d"}, timeout)
}

func run(ctx context.Context, projectDir string, files []string, command []string, timeout time.Duration) (Result, error) {
	if projectDir == "" || len(files) == 0 {
		return Result{}, fmt.Errorf("Compose 执行参数不完整")
	}
	args := []string{"compose", "--project-directory", projectDir}
	for _, file := range files {
		args = append(args, "-f", file)
	}
	args = append(args, command...)
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, "docker", args...)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	text := sanitizeOutput(output.String())
	if runCtx.Err() != nil {
		return Result{Output: text}, runCtx.Err()
	}
	if err != nil {
		return Result{Output: text}, fmt.Errorf("docker compose 执行失败: %w", err)
	}
	return Result{Output: text}, nil
}

func sanitizeOutput(output string) string {
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		for _, marker := range []string{"PASSWORD=", "TOKEN=", "SECRET=", "password:", "token:", "secret:"} {
			if index := strings.Index(strings.ToLower(line), strings.ToLower(marker)); index >= 0 {
				line = line[:index] + marker + "[REDACTED]"
			}
		}
		lines[i] = line
	}
	text := strings.Join(lines, "\n")
	if len(text) > 64*1024 {
		return text[:64*1024] + "..."
	}
	return text
}
