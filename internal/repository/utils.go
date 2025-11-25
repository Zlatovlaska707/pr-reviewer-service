package repository

import "errors"

// Общие ошибки репозитория.
var (
	ErrBuildQuery          = errors.New("failed to build SQL query")
	ErrExecuteQuery        = errors.New("failed to execute query")
	ErrScanResult          = errors.New("failed to scan result")
	ErrUserNotFound        = errors.New("user not found")
	ErrTeamNotFound        = errors.New("team not found")
	ErrPullRequestNotFound = errors.New("pull request not found")
)
