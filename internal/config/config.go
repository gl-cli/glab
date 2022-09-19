package config

import (
	"gitlab.com/gitlab-org/cli/pkg/prompt"
)

// PromptAndSetEnv : prompts user for value and returns default value if empty
func Prompt(question, defaultVal string) (envVal string, err error) {
	err = prompt.AskQuestionWithInput(&envVal, "config", question, defaultVal, false)
	if err != nil {
		return
	}
	return
}
