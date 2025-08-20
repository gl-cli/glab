package get_token

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	checkInterval = 50 * time.Millisecond
	infoInterval  = 2 * time.Second
	staleTimeout  = 1 * time.Minute
)

type fileMutex struct {
	filename string
	root     *os.Root
	f        *os.File
}

func withLock[T any](ctx context.Context, id string, fn func() (*T, error)) (t *T, err error) { //nolint:nonamedreturns
	root, err := lockFileBaseDir()
	if err != nil {
		return nil, err
	}

	fm := &fileMutex{filename: fmt.Sprintf("%s.lock", id), root: root}

	if err := fm.lock(ctx); err != nil {
		return nil, fmt.Errorf("failed to acquire file lock (%s): %w", filepath.Join(root.Name(), fm.filename), err)
	}
	defer func() {
		uerr := fm.unlock()
		if err == nil && uerr != nil {
			err = fmt.Errorf("failed to release file lock (%s): %w", filepath.Join(root.Name(), fm.filename), uerr)
		}
	}()

	return fn()
}

func lockFileBaseDir() (*os.Root, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}

	gitlabCacheDir := filepath.Join(cacheDir, "gitlab", "cli")
	err = os.MkdirAll(gitlabCacheDir, 0o700)
	if err != nil {
		return nil, err
	}

	return os.OpenRoot(gitlabCacheDir)
}

func (m *fileMutex) lock(ctx context.Context) error {
	t := time.NewTicker(checkInterval)
	defer t.Stop()
	infoT := time.NewTicker(infoInterval)
	defer infoT.Stop()

	done := ctx.Done()
	for {
		select {
		case <-done:
			return ctx.Err()
		case <-t.C:
			f, err := m.root.OpenFile(m.filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o400)
			if err != nil {
				info, err := m.root.Stat(m.filename)
				if err != nil {
					return fmt.Errorf("failed to check if lock file is stale: %w", err)
				}

				if time.Since(info.ModTime()) > staleTimeout {
					// remove lock file because it's stale
					if err := m.root.Remove(m.filename); err != nil {
						return fmt.Errorf("failed to remove stale lock file: %w", err)
					}
					fmt.Fprintf(os.Stderr, "Removed stale lock at %s from %s in order to re-use it\n", filepath.Join(m.root.Name(), m.filename), info.ModTime().Format(time.RFC3339))
				}
				continue
			}

			m.f = f
			return nil
		case <-infoT.C:
			fmt.Fprintln(os.Stderr, "Trying to acquire lock for token cache ...")
		}
	}
}

func (m *fileMutex) unlock() error {
	if m.f == nil {
		return errors.New("mutex is not acquired")
	}

	if err := m.f.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "unable to close mutex file at %s: %s\n", filepath.Join(m.root.Name(), m.filename), err)
	}

	if err := m.root.Remove(m.filename); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("unable to remove mutex file: %w", err)
	}

	m.f = nil
	return nil
}
