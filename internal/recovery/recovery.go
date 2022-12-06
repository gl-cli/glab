package recovery

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gitlab.com/gitlab-org/cli/internal/config"
)

const recoverDir = "recover"

func getRecoverDir(repoName string) (string, error) {
	configDir := config.ConfigDir()

	dir := filepath.Join(configDir, recoverDir, repoName)
	if config.CheckPathExists(dir) {
		return dir, nil
	}

	err := os.MkdirAll(dir, 0o755)
	if err != nil {
		return "", fmt.Errorf("creating recovery directory: %w", err)
	}

	return dir, nil
}

// CreateFile will create a filename under the recoverDir which lives inside
// the config.ConfigDir
func CreateFile(repoName, filename string, i any) (string, error) {
	dir, err := getRecoverDir(repoName)
	if err != nil {
		return "", err
	}

	fullPath := filepath.Join(dir, filename)
	f, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("creating recovery file: %w", err)
	}

	defer f.Close()

	if err := json.NewEncoder(f).Encode(i); err != nil {
		return "", fmt.Errorf("writing file: %w", err)
	}

	return fullPath, nil
}

// FromFile will try to open the filename and unmarshal the
// contents into a struct i of any type
func FromFile(repoName, fileName string, i any) error {
	dir, err := getRecoverDir(repoName)
	if err != nil {
		return err
	}

	fullPath := filepath.Join(dir, fileName)
	f, err := os.Open(fullPath)
	if err != nil {
		return err
	}

	if err := json.NewDecoder(f).Decode(&i); err != nil {
		return fmt.Errorf("could not decode %s into struct: %w", fileName, err)
	}

	// close and remove file
	f.Close()

	return os.Remove(fullPath)
}
