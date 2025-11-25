package teamdeactivate

import (
	"context"

	"pr-reviewer-service_Avito/internal/service"
)

type UseCase interface {
	MassDeactivate(ctx context.Context, input service.MassDeactivateInput) (service.MassDeactivateResult, error)
}
