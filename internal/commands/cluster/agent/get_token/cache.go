package get_token

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/zalando/go-keyring"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

const keyringService = "glab"

var (
	errNotFound            = errors.New("not found")
	errTokenExpired        = errors.New("token expired")
	errUnsupportedPlatform = errors.New("unsupported platform")
)

// storage defines the interface for token storage backends
type storage interface {
	get(id string) ([]byte, error)
	set(id string, data []byte) error
}

// keyringStorage implements storage using the system keyring
type keyringStorage struct{}

func (k *keyringStorage) get(id string) ([]byte, error) {
	data, err := keyring.Get(keyringService, id)
	switch err {
	case nil:
		return []byte(data), nil
	case keyring.ErrNotFound:
		return nil, errNotFound
	case keyring.ErrUnsupportedPlatform:
		return nil, errUnsupportedPlatform
	default:
		return nil, err
	}
}

func (k *keyringStorage) set(id string, data []byte) error {
	if err := keyring.Set(keyringService, id, string(data)); err != nil {
		if errors.Is(err, keyring.ErrUnsupportedPlatform) {
			return errUnsupportedPlatform
		}
		return err
	}
	return nil
}

// fileStorage implements storage using the filesystem
type fileStorage struct {
	root *os.Root
}

func newFileStorage() (*fileStorage, error) {
	cacheDir, err := userCacheDir()
	if err != nil {
		return nil, err
	}

	gitlabCacheDir := filepath.Join(cacheDir, "gitlab")
	err = os.MkdirAll(gitlabCacheDir, 0o700)
	if err != nil {
		return nil, err
	}

	root, err := os.OpenRoot(gitlabCacheDir)
	if err != nil {
		return nil, err
	}

	return &fileStorage{root: root}, nil
}

func (f *fileStorage) get(id string) ([]byte, error) {
	file, err := f.root.Open(id)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, errNotFound
		}
		return nil, err
	}
	defer file.Close()

	return io.ReadAll(file)
}

func (f *fileStorage) set(id string, data []byte) error {
	// TODO: handle race conditions, yes renaming a file is most often atomic, but not always
	// and it's unclear how it should be handled - should the content of the existing file be
	// read in case of a conflict? But what if the file is not yet fully written?
	// There are open questions that can be answered and implemented in a follow up iteration.
	file, err := f.root.OpenFile(id, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o400)
	if err != nil {
		return err
	}
	defer func() {
		cerr := file.Close()
		if err == nil {
			err = cerr
		}
	}()

	if _, err := file.Write(data); err != nil {
		return err
	}

	if err := file.Sync(); err != nil {
		return err
	}

	return nil
}

func (f *fileStorage) close() error {
	return f.root.Close()
}

type cache struct {
	id         string
	createFunc func() (*gitlab.PersonalAccessToken, error)
	storage    storage
}

func (c *cache) isTokenExpired(token *gitlab.PersonalAccessToken) bool {
	return time.Time(*token.ExpiresAt).Before(time.Now().UTC())
}

func (c *cache) get() (*gitlab.PersonalAccessToken, error) {
	token, err := c.getCachedToken()
	switch err {
	case nil:
		return token, nil
	case errNotFound:
		fallthrough
	case errTokenExpired:
		return c.createAndCacheToken()
	default:
		return nil, err
	}
}

func (c *cache) getCachedToken() (*gitlab.PersonalAccessToken, error) {
	data, err := c.storage.get(c.id)
	if err != nil {
		return nil, err
	}

	var token gitlab.PersonalAccessToken
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}

	if c.isTokenExpired(&token) {
		return nil, errTokenExpired
	}

	return &token, nil
}

func (c *cache) createAndCacheToken() (*gitlab.PersonalAccessToken, error) {
	token, err := c.createFunc()
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(token)
	if err != nil {
		return nil, err
	}

	if err := c.storage.set(c.id, data); err != nil {
		return nil, err
	}

	return token, nil
}

func userCacheDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		// On Windows, use %LOCALAPPDATA%
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			return localAppData, nil
		}
		// Fallback to %APPDATA%
		if appData := os.Getenv("APPDATA"); appData != "" {
			return appData, nil
		}
		return "", fmt.Errorf("neither LOCALAPPDATA nor APPDATA are defined")
	case "darwin":
		// On macOS, use ~/Library/Caches
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(homeDir, "Library", "Caches"), nil
	default:
		// On Linux and other Unix-like systems use XDG
		return xdgCacheDir()
	}
}

// xdgCacheDir returns the XDG cache directory
// Implemented according to https://specifications.freedesktop.org/basedir-spec/latest/
func xdgCacheDir() (string, error) {
	if xdgCache := os.Getenv("XDG_CACHE_HOME"); xdgCache != "" {
		return xdgCache, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to determine home directory to construct XDG cache directory")
	}

	return filepath.Join(homeDir, ".cache"), nil
}
