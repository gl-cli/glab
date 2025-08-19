package testing

//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -typed -destination=./git_runner.go -package=testing gitlab.com/gitlab-org/cli/internal/git GitRunner
