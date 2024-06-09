package utils

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"time"
)

func ExecuteCommand(cmdStr string, timeoutSecs ...int) (string, error) {
	var cmd *exec.Cmd
	timeout := 10 // default timeout in seconds stops certain commands from running indefinitely (i.e. ping)

	if len(timeoutSecs) > 0 { // simple check to see if a timeout was provided
		timeout = timeoutSecs[0]
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	switch runtime.GOOS {
	case "linux", "darwin":
		cmd = exec.CommandContext(ctx, "/bin/bash", "-c", cmdStr)
	case "windows":
		cmd = exec.CommandContext(ctx, "cmd.exe", "/C", cmdStr)
	default:
		return "", fmt.Errorf("unsupported platform")
	}

	// Get a pipe to read from standard output
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", err
	}

	var out bytes.Buffer
	done := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			line := scanner.Text()
			out.WriteString(line + "\n")
			//fmt.Println(line) // Optionally, print the line to stdout or handle it otherwise
		}
		done <- scanner.Err()
	}()

	select {
	case <-ctx.Done():
		// Timeout reached, kill the process and return what we have so far
		if killErr := cmd.Process.Kill(); killErr != nil {
			return fmt.Sprintf("%s\nCommand timed out after %d seconds", out.String(), timeout), nil
		}
		<-done // Allow goroutine to exit
		return fmt.Sprintf("%s\nCommand timed out after %d seconds", out.String(), timeout), nil
	case err := <-done:
		// Command finished
		if err != nil {
			return out.String(), err
		}
		if err := cmd.Wait(); err != nil {
			return out.String(), err
		}
	}

	return out.String(), nil
}
