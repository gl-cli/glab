package gittest

//go:generate go run go.uber.org/mock/mockgen@v0.5.2 -typed -destination=./git_runner.go -package=gittest gitlab.com/gitlab-org/cli/internal/git Git
