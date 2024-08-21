package save

import (
	"errors"
	"strings"

	"gitlab.com/gitlab-org/cli/commands/cmdutils"
)

// cleanDescription removes comments and trims space for all non-comment lines
func cleanDescription(message string) string {
	var sb strings.Builder
	for _, line := range strings.Split(message, "\n") {
		if strings.HasPrefix(line, "#") {
			continue
		}
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	return strings.TrimSpace(sb.String())
}

func promptForCommit(f *cmdutils.Factory, getText cmdutils.GetTextUsingEditor, defaultValue string) (string, error) {
	message := "\n# Please enter the commit message for this change. Lines starting\n# with '#' will be ignored. A message is required.\n#\n"
	editor, err := cmdutils.GetEditor(f.Config)
	if err != nil {
		return "", err
	}

	if defaultValue != "" {
		message = defaultValue + message
	}

	var cleanedDescription string
	if !f.IO.IsOutputTTY() {
		if defaultValue == "" {
			return "", errors.New("No commit message provided and no TTY. Please provide a commit message with the --message flag.")
		}
		return defaultValue, nil
	}
	description, err = getText(editor, "glab-stack-save-description*.gitcommit", message)
	if err != nil {
		return "", err
	}
	cleanedDescription = cleanDescription(description)

	if cleanedDescription == "" {
		return "", errors.New("Commit message cannot be empty.")
	}
	return cleanedDescription, nil
}
