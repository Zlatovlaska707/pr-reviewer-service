package usersetactivity

import (
	"context"

	"pr-reviewer-service_Avito/internal/domain"
)

type UseCase interface {
	SetUserActivity(ctx context.Context, userID string, active bool) (domain.User, error)
}
