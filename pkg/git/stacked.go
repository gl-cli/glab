package git

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gitlab.com/gitlab-org/cli/internal/run"
)

var StackLocation = filepath.Join(".git", "stacked")

type GitRunner interface {
	Git(args ...string) (string, error)
}

type StandardGitCommand struct{}

func (gitc StandardGitCommand) Git(args ...string) (string, error) {
	cmd := GitCommand(args...)
	output, err := run.PrepareCmd(cmd).Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}

func SetLocalConfig(key, value string) error {
	found, err := configValueExists(key, value)
	if err != nil {
		return fmt.Errorf("Git config value exists: %w", err)
	}

	if found {
		return nil
	}

	addCmd := GitCommand("config", "--local", key, value)
	_, err = run.PrepareCmd(addCmd).Output()
	if err != nil {
		return fmt.Errorf("setting local Git config: %w", err)
	}
	return nil
}

func GetCurrentStackTitle() (title string, err error) {
	title, err = Config("glab.currentstack")
	return
}

func AddStackRefDir(dir string) (string, error) {
	baseDir, err := ToplevelDir()
	if err != nil {
		return "", fmt.Errorf("finding top-level Git directory: %w", err)
	}

	createdDir := filepath.Join(baseDir, "/.git/refs/stacked/", dir)

	err = os.MkdirAll(createdDir, 0o755)
	if err != nil {
		return "", fmt.Errorf("creating stacked diff directory: %w", err)
	}

	return createdDir, nil
}

func StackRootDir(title string) (string, error) {
	baseDir, err := ToplevelDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(baseDir, StackLocation, title), nil
}

func AddStackRefFile(title string, stackRef StackRef) error {
	refDir, err := StackRootDir(title)
	if err != nil {
		return fmt.Errorf("error determining Git root: %v", err)
	}

	initialJsonData, err := json.Marshal(stackRef)
	if err != nil {
		return fmt.Errorf("error marshaling data: %v", err)
	}

	if _, err = os.Stat(refDir); os.IsNotExist(err) {
		err = os.MkdirAll(refDir, 0o700) // create directory if it doesn't exist
		if err != nil {
			return fmt.Errorf("error creating directory: %v", err)
		}
	}

	fullPath := filepath.Join(refDir, stackRef.SHA+".json")

	err = os.WriteFile(fullPath, initialJsonData, 0o644)
	if err != nil {
		return fmt.Errorf("error running writing file: %v", err)
	}

	return nil
}

func DeleteStackRefFile(title string, stackRef StackRef) error {
	refDir, err := StackRootDir(title)
	if err != nil {
		return fmt.Errorf("error determining Git root: %v", err)
	}

	fullPath := filepath.Join(refDir, stackRef.SHA+".json")

	err = os.Remove(fullPath)
	if err != nil {
		return fmt.Errorf("error removing stack file: %v", err)
	}

	return nil
}

func UpdateStackRefFile(title string, s StackRef) error {
	refDir, err := StackRootDir(title)
	if err != nil {
		return fmt.Errorf("error determining Git root: %v", err)
	}

	fullPath := filepath.Join(refDir, s.SHA+".json")

	initialJsonData, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("error marshaling data: %v", err)
	}

	err = os.WriteFile(fullPath, initialJsonData, 0o644)
	if err != nil {
		return fmt.Errorf("error writing file: %v", err)
	}

	return nil
}

func GetStacks() (stacks []Stack, err error) {
	topLevelDir, err := ToplevelDir()
	if err != nil {
		return nil, err
	}
	stackLocationDir := filepath.Join(topLevelDir, StackLocation)
	entries, err := os.ReadDir(stackLocationDir)
	if err != nil {
		return nil, err
	}
	for _, v := range entries {
		if !v.IsDir() {
			continue
		}
		stacks = append(stacks, Stack{Title: v.Name()})
	}
	return
}
