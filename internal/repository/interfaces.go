package repository

import (
	"context"

	"pr-reviewer-service_Avito/internal/domain"
)

// TransactionalRepository позволяет выполнять операции в транзакции.
type TransactionalRepository interface {
	WithTransaction(ctx context.Context, fn func(Repository) error) error
}

// Repository объединяет все доменные репозитории.
type Repository interface {
	TeamRepository
	UserRepository
	PRRepository
	StatsRepository
}

// TeamRepository содержит операции для работы с командами.
type TeamRepository interface {
	CreateTeam(ctx context.Context, team domain.Team) (domain.Team, error)
	GetTeam(ctx context.Context, teamName string) (domain.Team, error)
}

// UserRepository содержит операции для работы с пользователями.
type UserRepository interface {
	SetUserActivity(ctx context.Context, userID string, active bool) (domain.User, error)
	GetUserByID(ctx context.Context, userID string) (domain.User, error)
	ListActiveTeamMembers(ctx context.Context, teamName string, exclude []string) ([]domain.User, error)
	DeactivateUsers(ctx context.Context, userIDs []string) ([]domain.User, error)
}

// PRRepository содержит операции для работы с Pull Request'ами.
type PRRepository interface {
	CreatePullRequest(ctx context.Context, pr domain.PullRequest, reviewers []string) (domain.PullRequest, error)
	UpdatePRStatus(ctx context.Context, prID string, status domain.PRStatus) (domain.PullRequest, error)
	GetPullRequest(ctx context.Context, prID string) (domain.PullRequest, error)
	ReplaceReviewer(ctx context.Context, prID, oldReviewer, newReviewer, source string) (domain.PullRequest, string, error)
	ListReviewAssignments(ctx context.Context, userID string) ([]domain.PullRequestShort, error)
	ListOpenPRsByReviewer(ctx context.Context, reviewerIDs []string) (map[string][]string, error)
}

// StatsRepository содержит операции для получения статистики.
type StatsRepository interface {
	FetchAssignmentStats(ctx context.Context) (domain.AssignmentStats, error)
}

// HealthChecker описывает метод проверки соединения.
type HealthChecker interface {
	Ping(ctx context.Context) error
}
