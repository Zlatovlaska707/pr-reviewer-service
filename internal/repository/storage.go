package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"pr-reviewer-service_Avito/internal/domain"
	"pr-reviewer-service_Avito/internal/infrastructure/nower"
)

type pgxPool interface {
	Close()
	Ping(ctx context.Context) error
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Storage инкапсулирует работу с PostgreSQL.
type Storage struct {
	pool  pgxPool
	nower nower.Nower
	sb    squirrel.StatementBuilderType
}

// New создаёт новый слой хранения.
func New(pool pgxPool, nower nower.Nower) *Storage {
	return &Storage{
		pool:  pool,
		nower: nower,
		sb:    squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

// Close освобождает соединения пула.
func (s *Storage) Close() {
	s.pool.Close()
}

// Ping проверяет доступность подключения к БД.
func (s *Storage) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// WithTx оборачивает выполнение в транзакцию (низкоуровневый метод).
func (s *Storage) WithTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()
	if err := fn(tx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
}

// WithTransaction выполняет функцию fn в рамках транзакции на уровне репозитория.
func (s *Storage) WithTransaction(ctx context.Context, fn func(Repository) error) error {
	return s.WithTx(ctx, func(tx pgx.Tx) error {
		txRepo := newTxStorage(tx)
		return fn(txRepo)
	})
}

// CreateTeam создаёт новую команду и обновляет/создаёт пользователей в одной транзакции.
// Если команда уже существует, возвращает ошибку. Пользователи обновляются через UPSERT.
func (s *Storage) CreateTeam(ctx context.Context, team domain.Team) (domain.Team, error) {
	err := s.WithTx(ctx, func(tx pgx.Tx) error {
		// Проверка существования команды через Squirrel
		existsSQL, existsArgs, err := s.sb.
			Select("1").
			From("teams").
			Where(squirrel.Eq{"team_name": team.Name}).
			ToSql()
		if err != nil {
			slog.ErrorContext(ctx, "failed to build exists query", "error", err)
			return fmt.Errorf("%w: %v", ErrBuildQuery, err)
		}
		var exists bool
		if err := tx.QueryRow(ctx, "SELECT EXISTS("+existsSQL+")", existsArgs...).Scan(&exists); err != nil {
			slog.ErrorContext(ctx, "failed to check team existence", "error", err)
			return fmt.Errorf("%w: %v", ErrExecuteQuery, err)
		}
		if exists {
			return domain.ErrTeamExists
		}

		// Вставка команды
		insertTeamSQL, insertTeamArgs, err := s.sb.
			Insert("teams").
			Columns("team_name").
			Values(team.Name).
			ToSql()
		if err != nil {
			slog.ErrorContext(ctx, "failed to build insert team query", "error", err)
			return fmt.Errorf("%w: %v", ErrBuildQuery, err)
		}
		if _, err := tx.Exec(ctx, insertTeamSQL, insertTeamArgs...); err != nil {
			slog.ErrorContext(ctx, "failed to insert team", "error", err)
			return fmt.Errorf("%w: %v", ErrExecuteQuery, err)
		}

		// Вставка/обновление пользователей (UPSERT: если пользователь существует, обновляем его данные)
		now := s.nower.Now()
		for _, member := range team.Members {
			member.TeamName = team.Name
			insertUserSQL, insertUserArgs, err := s.sb.
				Insert("users").
				Columns("user_id", "username", "team_name", "is_active", "created_at", "updated_at").
				Values(member.ID, member.Username, member.TeamName, member.IsActive, now, now).
				Suffix("ON CONFLICT (user_id) DO UPDATE SET username=EXCLUDED.username, team_name=EXCLUDED.team_name, is_active=EXCLUDED.is_active, updated_at=EXCLUDED.updated_at").
				ToSql()
			if err != nil {
				slog.ErrorContext(ctx, "failed to build insert user query", "error", err)
				return fmt.Errorf("%w: %v", ErrBuildQuery, err)
			}
			if _, err := tx.Exec(ctx, insertUserSQL, insertUserArgs...); err != nil {
				slog.ErrorContext(ctx, "failed to insert/update user", "error", err, "user_id", member.ID)
				return fmt.Errorf("%w: %v", ErrExecuteQuery, err)
			}
		}
		return nil
	})
	if err != nil {
		return domain.Team{}, err
	}
	return s.GetTeam(ctx, team.Name)
}

// GetTeam возвращает команду с участниками.
func (s *Storage) GetTeam(ctx context.Context, teamName string) (domain.Team, error) {
	// Получение участников через Squirrel
	selectSQL, selectArgs, err := s.sb.
		Select("u.user_id", "u.username", "u.team_name", "u.is_active").
		From("users u").
		Where(squirrel.Eq{"u.team_name": teamName}).
		OrderBy("u.username ASC").
		ToSql()
	if err != nil {
		slog.ErrorContext(ctx, "failed to build select users query", "error", err)
		return domain.Team{}, fmt.Errorf("%w: %v", ErrBuildQuery, err)
	}

	rows, err := s.pool.Query(ctx, selectSQL, selectArgs...)
	if err != nil {
		slog.ErrorContext(ctx, "failed to query users", "error", err)
		return domain.Team{}, fmt.Errorf("%w: %v", ErrExecuteQuery, err)
	}
	defer rows.Close()

	var members []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Username, &u.TeamName, &u.IsActive); err != nil {
			slog.ErrorContext(ctx, "failed to scan user", "error", err)
			return domain.Team{}, fmt.Errorf("%w: %v", ErrScanResult, err)
		}
		members = append(members, u)
	}
	if err := rows.Err(); err != nil {
		return domain.Team{}, err
	}
	if len(members) == 0 {
		// Проверка существования команды
		existsSQL, existsArgs, err := s.sb.
			Select("1").
			From("teams").
			Where(squirrel.Eq{"team_name": teamName}).
			ToSql()
		if err != nil {
			return domain.Team{}, fmt.Errorf("%w: %v", ErrBuildQuery, err)
		}
		var exists bool
		if err := s.pool.QueryRow(ctx, "SELECT EXISTS("+existsSQL+")", existsArgs...).Scan(&exists); err != nil {
			return domain.Team{}, fmt.Errorf("%w: %v", ErrExecuteQuery, err)
		}
		if !exists {
			return domain.Team{}, domain.ErrTeamNotFound
		}
	}
	if members == nil {
		members = []domain.User{}
	}
	return domain.Team{Name: teamName, Members: members}, nil
}

// SetUserActivity обновляет флаг активности пользователя.
func (s *Storage) SetUserActivity(ctx context.Context, userID string, active bool) (domain.User, error) {
	now := s.nower.Now()
	updateSQL, updateArgs, err := s.sb.
		Update("users").
		Set("is_active", active).
		Set("updated_at", now).
		Where(squirrel.Eq{"user_id": userID}).
		ToSql()
	if err != nil {
		slog.ErrorContext(ctx, "failed to build update user query", "error", err)
		return domain.User{}, fmt.Errorf("%w: %v", ErrBuildQuery, err)
	}

	cmd, err := s.pool.Exec(ctx, updateSQL, updateArgs...)
	if err != nil {
		slog.ErrorContext(ctx, "failed to update user", "error", err)
		return domain.User{}, fmt.Errorf("%w: %v", ErrExecuteQuery, err)
	}
	if cmd.RowsAffected() == 0 {
		return domain.User{}, domain.ErrUserNotFound
	}
	return s.GetUserByID(ctx, userID)
}

// GetUserByID возвращает пользователя.
func (s *Storage) GetUserByID(ctx context.Context, userID string) (domain.User, error) {
	selectSQL, selectArgs, err := s.sb.
		Select("user_id", "username", "team_name", "is_active").
		From("users").
		Where(squirrel.Eq{"user_id": userID}).
		ToSql()
	if err != nil {
		slog.ErrorContext(ctx, "failed to build select user query", "error", err)
		return domain.User{}, fmt.Errorf("%w: %v", ErrBuildQuery, err)
	}

	var u domain.User
	err = s.pool.QueryRow(ctx, selectSQL, selectArgs...).Scan(&u.ID, &u.Username, &u.TeamName, &u.IsActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrUserNotFound
	}
	if err != nil {
		slog.ErrorContext(ctx, "failed to scan user", "error", err)
		return domain.User{}, fmt.Errorf("%w: %v", ErrScanResult, err)
	}
	return u, nil
}

// CreatePullRequest создаёт PR и возвращает его.
func (s *Storage) CreatePullRequest(ctx context.Context, pr domain.PullRequest, reviewers []string) (domain.PullRequest, error) {
	err := s.WithTx(ctx, func(tx pgx.Tx) error {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pull_requests WHERE pull_request_id=$1)`, pr.ID).Scan(&exists); err != nil {
			return err
		}
		if exists {
			return domain.ErrPRExists
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, created_at)
			VALUES ($1,$2,$3,$4,NOW())
		`, pr.ID, pr.Name, pr.AuthorID, string(pr.Status)); err != nil {
			return err
		}
		if len(reviewers) > 0 {
			if err := insertReviewers(ctx, tx, pr.ID, reviewers, "AUTO_ASSIGN"); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return domain.PullRequest{}, err
	}
	return s.GetPullRequest(ctx, pr.ID)
}

// insertReviewers добавляет ревьюверов к PR и создаёт события назначения в одном батче.
func insertReviewers(ctx context.Context, tx pgx.Tx, prID string, reviewers []string, source string) error {
	batch := &pgx.Batch{}
	for _, reviewer := range reviewers {
		// Добавляем связь PR-ревьювер
		batch.Queue(`
			INSERT INTO pull_request_reviewers (pull_request_id, reviewer_id, assigned_at)
			VALUES ($1,$2,NOW())
		`, prID, reviewer)
		// Создаём событие назначения для аудита
		batch.Queue(`
			INSERT INTO review_assignment_events (pull_request_id, reviewer_id, event_type, source)
			VALUES ($1,$2,'ASSIGNED',$3)
		`, prID, reviewer, source)
	}
	return tx.SendBatch(ctx, batch).Close()
}

// removeReviewer удаляет ревьювера из PR и создаёт событие снятия назначения.
func removeReviewer(ctx context.Context, tx pgx.Tx, prID, reviewerID, source string) error {
	batch := &pgx.Batch{}
	batch.Queue(`DELETE FROM pull_request_reviewers WHERE pull_request_id=$1 AND reviewer_id=$2`, prID, reviewerID)
	batch.Queue(`
		INSERT INTO review_assignment_events (pull_request_id, reviewer_id, event_type, source)
		VALUES ($1,$2,'UNASSIGNED',$3)
	`, prID, reviewerID, source)
	return tx.SendBatch(ctx, batch).Close()
}

// GetPullRequest возвращает полный PR.
func (s *Storage) GetPullRequest(ctx context.Context, prID string) (domain.PullRequest, error) {
	var pr domain.PullRequest
	var mergedAt *time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at
		FROM pull_requests WHERE pull_request_id=$1
	`, prID).Scan(&pr.ID, &pr.Name, &pr.AuthorID, &pr.Status, &pr.CreatedAt, &mergedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.PullRequest{}, domain.ErrPRNotFound
	}
	if err != nil {
		return domain.PullRequest{}, err
	}
	pr.MergedAt = mergedAt
	rows, err := s.pool.Query(ctx, `
		SELECT reviewer_id FROM pull_request_reviewers
		WHERE pull_request_id=$1
		ORDER BY reviewer_id ASC
	`, prID)
	if err != nil {
		return domain.PullRequest{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return domain.PullRequest{}, err
		}
		pr.AssignedReviewers = append(pr.AssignedReviewers, id)
	}
	if pr.AssignedReviewers == nil {
		pr.AssignedReviewers = []string{}
	}
	return pr, rows.Err()
}

// UpdatePRStatus устанавливает статус PR и merged_at (идемпотентная операция).
// Если PR уже имеет нужный статус, операция завершается без изменений.
func (s *Storage) UpdatePRStatus(ctx context.Context, prID string, status domain.PRStatus) (domain.PullRequest, error) {
	err := s.WithTx(ctx, func(tx pgx.Tx) error {
		var current domain.PRStatus
		var mergedAt *time.Time
		if err := tx.QueryRow(ctx, `
			SELECT status, merged_at FROM pull_requests WHERE pull_request_id=$1
		`, prID).Scan(&current, &mergedAt); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.ErrPRNotFound
			}
			return err
		}
		// Идемпотентность: если статус уже установлен, ничего не делаем
		if current == status && status == domain.PRStatusMerged && mergedAt != nil {
			return nil
		}
		// Устанавливаем merged_at только при переходе в статус MERGED
		if status == domain.PRStatusMerged {
			if _, err := tx.Exec(ctx, `
				UPDATE pull_requests SET status=$2, merged_at=COALESCE(merged_at, NOW())
				WHERE pull_request_id=$1
			`, prID, string(status)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return domain.PullRequest{}, err
	}
	return s.GetPullRequest(ctx, prID)
}

// ReplaceReviewer меняет одного ревьювера на другого.
func (s *Storage) ReplaceReviewer(ctx context.Context, prID, oldReviewer, newReviewer, source string) (domain.PullRequest, string, error) {
	err := s.WithTx(ctx, func(tx pgx.Tx) error {
		var status domain.PRStatus
		if err := tx.QueryRow(ctx, `SELECT status FROM pull_requests WHERE pull_request_id=$1`, prID).Scan(&status); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.ErrPRNotFound
			}
			return err
		}
		if status == domain.PRStatusMerged {
			return domain.ErrPRMerged
		}
		var assigned bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM pull_request_reviewers WHERE pull_request_id=$1 AND reviewer_id=$2
			)`, prID, oldReviewer).Scan(&assigned); err != nil {
			return err
		}
		if !assigned {
			return domain.ErrReviewerAbsent
		}
		if err := removeReviewer(ctx, tx, prID, oldReviewer, source); err != nil {
			return err
		}
		if newReviewer != "" {
			if err := insertReviewers(ctx, tx, prID, []string{newReviewer}, source); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return domain.PullRequest{}, "", err
	}
	pr, err := s.GetPullRequest(ctx, prID)
	return pr, newReviewer, err
}

// ListReviewAssignments возвращает PR'ы для ревьювера.
func (s *Storage) ListReviewAssignments(ctx context.Context, userID string) ([]domain.PullRequestShort, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT p.pull_request_id, p.pull_request_name, p.author_id, p.status
		FROM pull_request_reviewers r
		JOIN pull_requests p ON p.pull_request_id=r.pull_request_id
		WHERE r.reviewer_id=$1
		ORDER BY p.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.PullRequestShort
	for rows.Next() {
		var pr domain.PullRequestShort
		if err := rows.Scan(&pr.ID, &pr.Name, &pr.AuthorID, &pr.Status); err != nil {
			return nil, err
		}
		result = append(result, pr)
	}
	return result, rows.Err()
}

// ListActiveTeamMembers возвращает активных участников команды, исключая указанных пользователей.
// Использует динамическое построение SQL для фильтрации по списку исключений.
func (s *Storage) ListActiveTeamMembers(ctx context.Context, teamName string, exclude []string) ([]domain.User, error) {
	var params []any
	params = append(params, teamName)
	var where strings.Builder
	where.WriteString("team_name=$1 AND is_active=TRUE")
	// Динамически добавляем условие исключения пользователей
	if len(exclude) > 0 {
		ph := make([]string, len(exclude))
		for i, id := range exclude {
			params = append(params, id)
			ph[i] = fmt.Sprintf("$%d", i+2)
		}
		where.WriteString(" AND user_id NOT IN (" + strings.Join(ph, ",") + ")")
	}
	query := fmt.Sprintf(`SELECT user_id, username, team_name, is_active FROM users WHERE %s`, where.String())
	rows, err := s.pool.Query(ctx, query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Username, &u.TeamName, &u.IsActive); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// DeactivateUsers деактивирует список пользователей и возвращает обновлённых.
func (s *Storage) DeactivateUsers(ctx context.Context, userIDs []string) ([]domain.User, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	err := s.WithTx(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE users SET is_active=FALSE, updated_at=NOW()
			WHERE user_id = ANY($1)
		`, userIDs)
		return err
	})
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT user_id, username, team_name, is_active FROM users WHERE user_id = ANY($1)
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Username, &u.TeamName, &u.IsActive); err != nil {
			return nil, err
		}
		result = append(result, u)
	}
	return result, rows.Err()
}

// ListOpenPRsByReviewer возвращает открытые PR по ревьюверам.
func (s *Storage) ListOpenPRsByReviewer(ctx context.Context, reviewerIDs []string) (map[string][]string, error) {
	if len(reviewerIDs) == 0 {
		return map[string][]string{}, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT r.reviewer_id, r.pull_request_id
		FROM pull_request_reviewers r
		JOIN pull_requests p ON p.pull_request_id=r.pull_request_id
		WHERE r.reviewer_id = ANY($1) AND p.status='OPEN'
	`, reviewerIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string][]string)
	for rows.Next() {
		var reviewerID, prID string
		if err := rows.Scan(&reviewerID, &prID); err != nil {
			return nil, err
		}
		result[reviewerID] = append(result[reviewerID], prID)
	}
	return result, rows.Err()
}

// FetchAssignmentStats собирает статистику.
func (s *Storage) FetchAssignmentStats(ctx context.Context) (domain.AssignmentStats, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT u.user_id, u.username, u.team_name,
		       COUNT(r.pull_request_id) AS assigned_total,
		       SUM(CASE WHEN p.status='OPEN' THEN 1 ELSE 0 END) AS active_pull_requests
		FROM users u
		LEFT JOIN pull_request_reviewers r ON r.reviewer_id=u.user_id
		LEFT JOIN pull_requests p ON p.pull_request_id=r.pull_request_id
		GROUP BY u.user_id, u.username, u.team_name
		ORDER BY assigned_total DESC
	`)
	if err != nil {
		return domain.AssignmentStats{}, err
	}
	defer rows.Close()
	var perUser []domain.UserAssignmentStat
	for rows.Next() {
		var stat domain.UserAssignmentStat
		if err := rows.Scan(&stat.UserID, &stat.Username, &stat.TeamName, &stat.Assigned, &stat.ActivePRs); err != nil {
			return domain.AssignmentStats{}, err
		}
		perUser = append(perUser, stat)
	}
	if err := rows.Err(); err != nil {
		return domain.AssignmentStats{}, err
	}
	rows2, err := s.pool.Query(ctx, `
		SELECT p.pull_request_id, COUNT(r.reviewer_id) AS reviewer_count
		FROM pull_requests p
		LEFT JOIN pull_request_reviewers r ON r.pull_request_id=p.pull_request_id
		GROUP BY p.pull_request_id
		ORDER BY reviewer_count DESC
	`)
	if err != nil {
		return domain.AssignmentStats{}, err
	}
	defer rows2.Close()
	var perPR []domain.PRAssignmentStat
	for rows2.Next() {
		var stat domain.PRAssignmentStat
		if err := rows2.Scan(&stat.PullRequestID, &stat.ReviewerCount); err != nil {
			return domain.AssignmentStats{}, err
		}
		perPR = append(perPR, stat)
	}
	return domain.AssignmentStats{PerUser: perUser, PerPR: perPR}, rows2.Err()
}
