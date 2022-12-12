package flag

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func GroupOverride(cmd *cobra.Command) (string, error) {
	// Bind cobra command persistent flags to env user viper
	viper.SetEnvPrefix("GITLAB")
	err := viper.BindEnv("group")
	if err != nil {
		return "", err
	}

	// Get group from env
	groupFromEnv := viper.GetString("group")

	// Get group/repo flags
	group, err := cmd.Flags().GetString("group")
	if err != nil {
		return "", err
	}
	repo, err := cmd.Flags().GetString("repo")
	if err != nil {
		return "", err
	}

	// Determine which group to use based on env and repo/group flags
	switch {
	case repo != "":
		// Ignore any groups if the repo flag is set
		return "", nil
	case group != "":
		// Use the group flag if set and repo flag is not set
		return group, nil
	default:
		// Consider environment if no repo or group flags are set
		return groupFromEnv, nil
	}
}
