package utils

import (
	"fmt"
	"os"
	"os/exec"
)

// ValidateExecutable checks if the specified executable exists and is accessible.
func ValidateExecutable(name string) error {
	_, err := exec.LookPath(name)
	if err != nil {
		return fmt.Errorf("%s executable not found in PATH: %w", name, err)
	}
	return nil
}

// ExtractArgs extracts arguments after the specified executable name from os.Args.
func ExtractArgs(name string) ([]string, error) {
	osArgs := os.Args

	// Find the index where name appears in the arguments
	executableIndex := -1
	for i, arg := range osArgs {
		if arg == name {
			executableIndex = i
			break
		}
	}

	if executableIndex == -1 {
		return nil, fmt.Errorf("could not find '%s' in command arguments", name)
	}

	// Return all arguments after executable
	return osArgs[executableIndex+1:], nil
}
