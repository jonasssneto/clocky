//go:build !windows

package updater

import (
	"fmt"
	"os"
	"syscall"
)

var startProcess = func(executable string, arguments []string) error {
	return syscall.Exec(executable, arguments, os.Environ()) //nolint:gosec // executable is the current Clocky binary.
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
