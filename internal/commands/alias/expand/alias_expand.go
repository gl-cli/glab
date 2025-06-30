package expand

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"gitlab.com/gitlab-org/cli/pkg/execext"

	"github.com/google/shlex"
	"gitlab.com/gitlab-org/cli/internal/config"
)

// ExpandAlias processes argv to see if it should be rewritten according to a user's aliases. The
// second return value indicates whether the alias should be executed in a new shell process instead
// of running gh itself.
func ExpandAlias(cfg config.Config, args []string, findShFunc func() (string, error)) (expanded []string, isShell bool, err error) {
	if len(args) < 2 {
		// the command is lacking a subcommand
		return
	}
	expanded = args[1:]

	aliases, err := cfg.Aliases()
	if err != nil {
		return
	}

	expansion, ok := aliases.Get(args[1])
	if !ok {
		return
	}

	if strings.HasPrefix(expansion, "!") {
		isShell = true
		if findShFunc == nil {
			findShFunc = findSh
		}
		shPath, shErr := findShFunc()
		if shErr != nil {
			err = shErr
			return
		}

		expanded = []string{shPath, "-c", expansion[1:]}

		if len(args[2:]) > 0 {
			expanded = append(expanded, "--")
			expanded = append(expanded, args[2:]...)
		}

		return
	}

	var extraArgs []string
	for i, a := range args[2:] {
		if !strings.Contains(expansion, "$") {
			extraArgs = append(extraArgs, a)
		} else {
			expansion = strings.ReplaceAll(expansion, fmt.Sprintf("$%d", i+1), a)
		}
	}
	lingeringRE := regexp.MustCompile(`\$\d`)
	if lingeringRE.MatchString(expansion) {
		err = fmt.Errorf("not enough arguments for alias: %s", expansion)
		return
	}

	var newArgs []string
	newArgs, err = shlex.Split(expansion)
	if err != nil {
		return
	}

	expanded = append(newArgs, extraArgs...)
	return
}

func findSh() (string, error) {
	shPath, err := execext.LookPath("sh")
	if err == nil {
		return shPath, nil
	}

	if runtime.GOOS == "windows" {
		winNotFoundErr := errors.New("unable to locate sh to execute the shell alias with. The sh.exe interpreter is typically distributed with Git for Windows")
		// We can try and find a sh executable in a Git for Windows install
		gitPath, err := execext.LookPath("git")
		if err != nil {
			return "", winNotFoundErr
		}

		shPath = filepath.Join(filepath.Dir(gitPath), "..", "bin", "sh.exe")
		_, err = os.Stat(shPath)
		if err != nil {
			return "", winNotFoundErr
		}

		return shPath, nil
	}

	return "", errors.New("unable to locate sh to execute shell alias with")
}
