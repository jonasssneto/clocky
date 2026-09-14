//go:build windows

package updater

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

var startProcess = func(executable string, arguments []string) error {
	command := exec.CommandContext(context.Background(), executable, arguments[1:]...) //nolint:gosec // executable is the current Clocky binary.
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Start()
}

func RestartCurrentProcess() error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate Clocky executable: %w", err)
	}
	arguments := append([]string{executable}, os.Args[1:]...)
	if err := startProcess(executable, arguments); err != nil {
		return fmt.Errorf("restart Clocky: %w", err)
	}
	return nil
}
