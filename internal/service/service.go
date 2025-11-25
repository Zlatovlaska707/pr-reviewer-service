package service

import (
	"context"
	"strings"
	"time"

	trm "github.com/avito-tech/go-transaction-manager/trm/v2"

	"pr-reviewer-service_Avito/internal/config"
	"pr-reviewer-service_Avito/internal/domain"
	"pr-reviewer-service_Avito/internal/infrastructure/randomizer"
	"pr-reviewer-service_Avito/internal/metrics"
	"pr-reviewer-service_Avito/internal/repository"
)

const (
	// DefaultOperationTimeout таймаут по умолчанию для обычных операций
	DefaultOperationTimeout = 30 * time.Second
	// DefaultLongOperationTimeout таймаут по умолчанию для длительных операций
	DefaultLongOperationTimeout = 60 * time.Second
)

// Repository описывает операции, которые требуются сервису.
type Repository interface {
	repository.Repository
}

// Service агрегирует бизнес-логику приложения.
type Service struct {
	repo       Repository
	health     repository.HealthChecker
	cfg        config.Config
	trMgr      trm.Manager
	randomizer randomizer.Randomizer
}

func New(repo Repository, cfg config.Config, trMgr trm.Manager, randomizer randomizer.Randomizer) *Service {
	svc := &Service{
		repo:       repo,
		cfg:        cfg,
		trMgr:      trMgr,
		randomizer: randomizer,
	}
	if svc.cfg.Timeouts.Operation <= 0 {
		svc.cfg.Timeouts.Operation = DefaultOperationTimeout
	}
	if svc.cfg.Timeouts.LongOperation <= 0 {
		svc.cfg.Timeouts.LongOperation = DefaultLongOperationTimeout
	}
	if checker, ok := repo.(repository.HealthChecker); ok {
		svc.health = checker
	}
	return svc
}

// CreateTeam обрабатывает создание команды.
func (s *Service) CreateTeam(ctx context.Context, team domain.Team) (domain.Team, error) {
	ctx, cancel := s.shortOperationContext(ctx)
	defer cancel()

	if err := ValidateTeamName(team.Name); err != nil {
		return domain.Team{}, err
	}
	for _, member := range team.Members {
		if err := ValidateUserID(member.ID); err != nil {
			return domain.Team{}, err
		}
	}
	created, err := s.repo.CreateTeam(ctx, team)
	if err == nil {
		metrics.IncTeamsCreated()
		metrics.AddUsersProcessed(len(created.Members))
	}
	return created, err
}

// GetTeam возвращает команду.
func (s *Service) GetTeam(ctx context.Context, name string) (domain.Team, error) {
	ctx, cancel := s.shortOperationContext(ctx)
	defer cancel()

	if err := ValidateTeamName(name); err != nil {
		return domain.Team{}, err
	}
	return s.repo.GetTeam(ctx, name)
}

// SetUserActivity обновляет флаг активности.
func (s *Service) SetUserActivity(ctx context.Context, userID string, active bool) (domain.User, error) {
	ctx, cancel := s.shortOperationContext(ctx)
	defer cancel()

	if err := ValidateUserID(userID); err != nil {
		return domain.User{}, err
	}
	return s.repo.SetUserActivity(ctx, userID, active)
}

// CreatePullRequest создаёт PR и автоматически назначает до 2 случайных ревьюверов из команды автора.
func (s *Service) CreatePullRequest(ctx context.Context, prID, name, authorID string) (domain.PullRequest, error) {
	ctx, cancel := s.shortOperationContext(ctx)
	defer cancel()

	if err := ValidatePRID(prID); err != nil {
		return domain.PullRequest{}, err
	}
	if err := ValidatePRName(name); err != nil {
		return domain.PullRequest{}, err
	}
	if err := ValidateUserID(authorID); err != nil {
		return domain.PullRequest{}, err
	}
	author, err := s.repo.GetUserByID(ctx, authorID)
	if err != nil {
		return domain.PullRequest{}, err
	}
	// Исключаем автора из списка кандидатов на ревью
	candidates, err := s.repo.ListActiveTeamMembers(ctx, author.TeamName, []string{author.ID})
	if err != nil {
		return domain.PullRequest{}, err
	}
	// Выбираем случайных ревьюверов (до 2 штук)
	reviewers := pickRandomIDs(candidates, 2, s.randomizer)
	pr := domain.PullRequest{
		ID:        prID,
		Name:      name,
		AuthorID:  author.ID,
		Status:    domain.PRStatusOpen,
		CreatedAt: time.Now(),
	}
	created, err := s.repo.CreatePullRequest(ctx, pr, reviewers)
	if err == nil {
		metrics.IncPullRequestsCreated()
	}
	return created, err
}

// MergePullRequest помечает PR как MERGED.
func (s *Service) MergePullRequest(ctx context.Context, prID string) (domain.PullRequest, error) {
	ctx, cancel := s.shortOperationContext(ctx)
	defer cancel()

	if err := ValidatePRID(prID); err != nil {
		return domain.PullRequest{}, err
	}
	return s.repo.UpdatePRStatus(ctx, prID, domain.PRStatusMerged)
}

// ReassignReviewer переназначает ревьювера на случайного активного участника из той же команды.
// Исключает автора PR и всех уже назначенных ревьюверов.
func (s *Service) ReassignReviewer(ctx context.Context, prID, oldReviewer string) (domain.PullRequest, string, error) {
	ctx, cancel := s.shortOperationContext(ctx)
	defer cancel()

	if err := ValidatePRID(prID); err != nil {
		return domain.PullRequest{}, "", err
	}
	if err := ValidateUserID(oldReviewer); err != nil {
		return domain.PullRequest{}, "", err
	}
	pr, err := s.repo.GetPullRequest(ctx, prID)
	if err != nil {
		return domain.PullRequest{}, "", err
	}
	// Нельзя переназначать ревьюверов для уже смерженных PR
	if pr.Status == domain.PRStatusMerged {
		return domain.PullRequest{}, "", domain.ErrPRMerged
	}
	// Проверяем, что старый ревьювер действительно назначен на PR
	var assigned bool
	for _, reviewer := range pr.AssignedReviewers {
		if reviewer == oldReviewer {
			assigned = true
			break
		}
	}
	if !assigned {
		return domain.PullRequest{}, "", domain.ErrReviewerAbsent
	}
	oldUser, err := s.repo.GetUserByID(ctx, oldReviewer)
	if err != nil {
		return domain.PullRequest{}, "", err
	}
	// Исключаем старого ревьювера, автора и всех остальных назначенных ревьюверов
	exclude := append([]string{oldReviewer, pr.AuthorID}, pr.AssignedReviewers...)
	candidates, err := s.repo.ListActiveTeamMembers(ctx, oldUser.TeamName, uniqueIDs(exclude))
	if err != nil {
		return domain.PullRequest{}, "", err
	}
	// Если нет доступных кандидатов в команде, возвращаем ошибку
	if len(candidates) == 0 {
		return domain.PullRequest{}, "", domain.ErrNoCandidate
	}
	// Выбираем случайного нового ревьювера
	newReviewer := pickRandomIDs(candidates, 1, s.randomizer)
	pr, replacedBy, err := s.repo.ReplaceReviewer(ctx, prID, oldReviewer, newReviewer[0], "MANUAL_REASSIGN")
	if err == nil {
		metrics.IncReassignments()
	}
	return pr, replacedBy, err
}

// ListReviewAssignments возвращает PR пользователя.
func (s *Service) ListReviewAssignments(ctx context.Context, userID string) ([]domain.PullRequestShort, error) {
	ctx, cancel := s.shortOperationContext(ctx)
	defer cancel()

	if err := ValidateUserID(userID); err != nil {
		return nil, err
	}
	_, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListReviewAssignments(ctx, userID)
}

// Stats возвращает агрегаты.
func (s *Service) Stats(ctx context.Context) (domain.AssignmentStats, error) {
	ctx, cancel := s.shortOperationContext(ctx)
	defer cancel()

	return s.repo.FetchAssignmentStats(ctx)
}

// MassDeactivateInput описывает вход для массовой деактивации.
type MassDeactivateInput struct {
	TeamName string
	UserIDs  []string
}

// MassDeactivateResult содержит результат операции.
type MassDeactivateResult struct {
	Deactivated []domain.User       `json:"deactivated"`
	Reassigned  map[string][]string `json:"reassignments"` // userID -> список PR
	Skipped     map[string]string   `json:"skipped"`       // userID -> причина
}

// MassDeactivate деактивирует пользователей команды и безопасно переназначает их PR на других ревьюверов.
func (s *Service) MassDeactivate(ctx context.Context, input MassDeactivateInput) (MassDeactivateResult, error) {
	ctx, cancel := s.longOperationContext(ctx)
	defer cancel()

	// Валидация входных данных
	if err := ValidateTeamName(input.TeamName); err != nil {
		return MassDeactivateResult{}, err
	}
	for _, userID := range input.UserIDs {
		if err := ValidateUserID(userID); err != nil {
			return MassDeactivateResult{}, err
		}
	}

	// Получаем команду вне транзакции для валидации
	team, err := s.repo.GetTeam(ctx, input.TeamName)
	if err != nil {
		return MassDeactivateResult{}, err
	}

	// Подготовка списка пользователей для деактивации
	// Если список пуст, деактивируем всех активных участников команды
	membership := make(map[string]domain.User, len(team.Members))
	for _, member := range team.Members {
		membership[member.ID] = member
	}
	targetIDs := input.UserIDs
	if len(targetIDs) == 0 {
		// Деактивируем всех активных участников
		targetIDs = make([]string, 0, len(team.Members))
		for _, m := range team.Members {
			if m.IsActive {
				targetIDs = append(targetIDs, m.ID)
			}
		}
	} else {
		// Фильтруем только активных пользователей из указанного списка
		filtered := targetIDs[:0]
		for _, id := range targetIDs {
			member, ok := membership[id]
			if !ok {
				return MassDeactivateResult{}, domain.ErrUserNotFound
			}
			if !member.IsActive {
				continue
			}
			filtered = append(filtered, id)
		}
		targetIDs = filtered
	}
	if len(targetIDs) == 0 {
		return MassDeactivateResult{}, nil
	}

	// Выполняем всю операцию в одной транзакции через transaction manager
	var result MassDeactivateResult
	err = s.trMgr.Do(ctx, func(ctx context.Context) error {
		// Деактивируем пользователей
		deactivated, err := s.repo.DeactivateUsers(ctx, targetIDs)
		if err != nil {
			return err
		}

		// Получаем открытые PR для деактивированных пользователей
		openPRs, err := s.repo.ListOpenPRsByReviewer(ctx, targetIDs)
		if err != nil {
			return err
		}

		result = MassDeactivateResult{
			Deactivated: deactivated,
			Reassigned:  map[string][]string{},
			Skipped:     map[string]string{},
		}

		// Переназначаем ревьюверов для каждого PR деактивированного пользователя
		for _, userID := range targetIDs {
			prList := openPRs[userID]
			if len(prList) == 0 {
				continue
			}
			for _, prID := range prList {
				prItem, err := s.repo.GetPullRequest(ctx, prID)
				if err != nil {
					result.Skipped[userID] = err.Error()
					break
				}
				// Исключаем деактивированного пользователя, автора и всех остальных ревьюверов
				exclude := append([]string{userID, prItem.AuthorID}, prItem.AssignedReviewers...)
				candidates, err := s.repo.ListActiveTeamMembers(ctx, team.Name, uniqueIDs(exclude))
				if err != nil {
					result.Skipped[userID] = err.Error()
					break
				}
				// Если есть кандидаты, выбираем случайного; иначе оставляем PR без ревьювера
				var newReviewer string
				if len(candidates) > 0 {
					newReviewer = pickRandomIDs(candidates, 1, s.randomizer)[0]
				}
				if _, _, err := s.repo.ReplaceReviewer(ctx, prID, userID, newReviewer, "TEAM_DEACTIVATION"); err != nil {
					result.Skipped[userID] = err.Error()
					break
				}
				metrics.IncReassignments()
				result.Reassigned[userID] = append(result.Reassigned[userID], prID)
			}
		}

		if result.Deactivated == nil {
			result.Deactivated = []domain.User{}
		}
		return nil
	})

	if err != nil {
		return MassDeactivateResult{}, err
	}

	return result, nil
}

// HealthCheck возвращает состояние зависимостей сервиса.
func (s *Service) HealthCheck(ctx context.Context) error {
	if s.health == nil {
		return nil
	}
	ctx, cancel := s.shortOperationContext(ctx)
	defer cancel()
	return s.health.Ping(ctx)
}

// shortOperationContext создаёт контекст с таймаутом для обычных операций.
func (s *Service) shortOperationContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, s.cfg.Timeouts.Operation)
}

// longOperationContext создаёт контекст с таймаутом для длительных операций.
func (s *Service) longOperationContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, s.cfg.Timeouts.LongOperation)
}

// pickRandomIDs выбирает случайных пользователей из списка (до limit штук) используя Fisher-Yates shuffle.
func pickRandomIDs(users []domain.User, limit int, randomizer randomizer.Randomizer) []string {
	if len(users) == 0 || limit <= 0 {
		return nil
	}
	ids := make([]string, len(users))
	for i, u := range users {
		ids[i] = u.ID
	}
	// Перемешиваем весь список
	randomizer.Shuffle(len(ids), func(i, j int) {
		ids[i], ids[j] = ids[j], ids[i]
	})
	// Если элементов меньше или равно limit, возвращаем все
	if len(ids) <= limit {
		return ids
	}
	// Иначе возвращаем первые limit элементов
	return ids[:limit]
}

// uniqueIDs удаляет дубликаты и пустые строки из списка.
func uniqueIDs(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, v := range values {
		if strings.TrimSpace(v) == "" {
			continue
		}
		set[v] = struct{}{}
	}
	res := make([]string, 0, len(set))
	for k := range set {
		res = append(res, k)
	}
	return res
}
