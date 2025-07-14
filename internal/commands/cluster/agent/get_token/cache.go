package get_token

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type cache struct {
	dir        *os.Root
	prefix     string
	createFunc func() (*gitlab.PersonalAccessToken, error)
}

func (c *cache) get() (*gitlab.PersonalAccessToken, error) {
	pat, err := c.read()
	if err != nil {
		return nil, err
	}

	if pat == nil {
		pat, err = c.createFunc()
		if err != nil {
			return nil, err
		}

		if err := c.write(pat); err != nil {
			return nil, err
		}
	}

	return pat, nil
}

func (c *cache) read() (*gitlab.PersonalAccessToken, error) {
	var pat *gitlab.PersonalAccessToken
	err := filepath.WalkDir(c.dir.Name(), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		name := d.Name()

		suffix, found := strings.CutPrefix(name, c.prefix)
		if !found {
			return nil
		}

		ts, err := strconv.ParseInt(suffix, 10, 64)
		if err != nil {
			return err
		}
		expiresAt := time.Unix(ts, 0)
		if expiresAt.Before(time.Now().UTC()) {
			_ = c.dir.Remove(name) // I don't think we care about the error, do we?
			return nil
		}

		// token seems valid looking at the expiration date
		f, err := c.dir.Open(name)
		if err != nil {
			return err
		}
		defer f.Close()

		token, err := io.ReadAll(f)
		if err != nil {
			return err
		}

		pat = &gitlab.PersonalAccessToken{
			Token:     string(token),
			ExpiresAt: gitlab.Ptr(gitlab.ISOTime(expiresAt)),

			// NOTE: the other fields are unused, so we don't need to populate them.
		}
		return fs.SkipAll
	})
	return pat, err
}

func (c *cache) write(pat *gitlab.PersonalAccessToken) (err error) {
	// TODO: handle race conditions, yes renaming a file is most often atomic, but not always
	// and it's unclear how it should be handled - should the content of the existing file be
	// read in case of a conflict? But what if the file is not yet fully written?
	// There are open questions that can be answered and implemented in a follow up iteration.
	f, err := c.dir.OpenFile(fmt.Sprintf("%s%d", c.prefix, time.Time(*pat.ExpiresAt).Unix()), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o400)
	if err != nil {
		return err
	}
	defer func() {
		cerr := f.Close()
		if err == nil {
			err = cerr
		}
	}()

	if _, err := f.WriteString(pat.Token); err != nil {
		return err
	}

	if err := f.Sync(); err != nil {
		return err
	}

	return nil
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
