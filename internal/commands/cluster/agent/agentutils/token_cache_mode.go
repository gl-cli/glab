package agentutils

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
)

const (
	DefaultCacheMode          CacheMode = "default"
	ForcedKeyringCacheMode    CacheMode = "force-keyring"
	ForcedFilesystemCacheMode CacheMode = "force-filesystem"
	NoCacheCacheMode          CacheMode = "no"
)

var CacheModes = []CacheMode{DefaultCacheMode, ForcedKeyringCacheMode, ForcedFilesystemCacheMode, NoCacheCacheMode}

type CacheMode = string

func AddTokenCacheModeFlag(fl *pflag.FlagSet, f *string) {
	fl.VarP(cmdutils.NewEnumValue(CacheModes, DefaultCacheMode, f), "cache-mode", "c", fmt.Sprintf("Mode to use for caching the token (allowed: %s)", strings.Join(CacheModes, ", ")))
}
