package preflightfile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func FindFile(startDir, explicitPath string) (string, error) {
	if explicitPath != "" {
		if _, err := os.Stat(explicitPath); err != nil {
			return "", fmt.Errorf("preflight file not found: %w", err)
		}
		return explicitPath, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	currentDir, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	for {
		preflightPath := filepath.Join(currentDir, ".preflight")
		if _, err := os.Stat(preflightPath); err == nil {
			return preflightPath, nil
		}

		if currentDir == homeDir {
			break
		}

		gitPath := filepath.Join(currentDir, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			break
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			// Reached filesystem root
			break
		}
		currentDir = parentDir
	}

	return "", errors.New(".preflight file not found")
}

func ParseFile(path string) ([]string, error) {
	data, err := os.ReadFile(path) //nolint:gosec // intentional: reading .preflight file
	if err != nil {
		return nil, fmt.Errorf("failed to read preflight file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	commands := []string{}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		if !strings.HasPrefix(trimmed, "preflight") {
			trimmed = "preflight " + trimmed
		}

		commands = append(commands, trimmed)
	}

	return commands, nil
}
