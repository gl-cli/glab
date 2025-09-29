package dbg

import (
	"log"

	"gitlab.com/gitlab-org/cli/internal/utils"
)

func Debug(output ...string) {
	if enabled, found := utils.IsEnvVarEnabled("DEBUG"); found && enabled {
		log.Print(output)
	}

	if enabled, found := utils.IsEnvVarEnabled("GLAB_DEBUG"); found && enabled {
		log.Print(output)
	}
}

func Debugf(format string, v ...any) {
	if enabled, found := utils.IsEnvVarEnabled("DEBUG"); found && enabled {
		log.Printf(format, v...)
	}

	if enabled, found := utils.IsEnvVarEnabled("GLAB_DEBUG"); found && enabled {
		log.Printf(format, v...)
	}
}
