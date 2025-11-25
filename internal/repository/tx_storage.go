package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"pr-reviewer-service_Avito/internal/domain"
)

// txStorage обёртка над транзакцией, реализующая интерфейс Repository.
type txStorage struct {
	tx pgx.Tx
}

func newTxStorage(tx pgx.Tx) *txStorage {
	return &txStorage{tx: tx}
}

// CreateTeam создаёт новую команду и апдейтит пользователей.
func (s *txStorage) CreateTeam(ctx context.Context, team domain.Team) (domain.Team, error) {
	var exists bool
	if err := s.tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM teams WHERE team_name=$1)`, team.Name).Scan(&exists); err != nil {
		return domain.Team{}, err
	}
	if exists {
		return domain.Team{}, domain.ErrTeamExists
	}
	if _, err := s.tx.Exec(ctx, `INSERT INTO teams (team_name) VALUES ($1)`, team.Name); err != nil {
		return domain.Team{}, err
	}
	for _, member := range team.Members {
		member.TeamName = team.Name
		if _, err := s.tx.Exec(ctx, `
			INSERT INTO users (user_id, username, team_name, is_active, created_at, updated_at)
			VALUES ($1,$2,$3,$4,NOW(),NOW())
			ON CONFLICT (user_id) DO UPDATE
			SET username=EXCLUDED.username,
			    team_name=EXCLUDED.team_name,
			    is_active=EXCLUDED.is_active,
			    updated_at=NOW()
		`, member.ID, member.Username, member.TeamName, member.IsActive); err != nil {
			return domain.Team{}, err
		}
	}
	return s.GetTeam(ctx, team.Name)
}

// GetTeam возвращает команду с участниками.
func (s *txStorage) GetTeam(ctx context.Context, teamName string) (domain.Team, error) {
	rows, err := s.tx.Query(ctx, `
		SELECT u.user_id, u.username, u.team_name, u.is_active
		FROM users u
		WHERE u.team_name=$1
		ORDER BY u.username ASC
	`, teamName)
	if err != nil {
		return domain.Team{}, err
	}
	defer rows.Close()

	var members []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Username, &u.TeamName, &u.IsActive); err != nil {
			return domain.Team{}, err
		}
		members = append(members, u)
	}
	if err := rows.Err(); err != nil {
		return domain.Team{}, err
	}
	if len(members) == 0 {
		var exists bool
		if err := s.tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM teams WHERE team_name=$1)`, teamName).Scan(&exists); err != nil {
			return domain.Team{}, err
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
func (s *txStorage) SetUserActivity(ctx context.Context, userID string, active bool) (domain.User, error) {
	cmd, err := s.tx.Exec(ctx, `UPDATE users SET is_active=$2, updated_at=NOW() WHERE user_id=$1`, userID, active)
	if err != nil {
		return domain.User{}, err
	}
	if cmd.RowsAffected() == 0 {
		return domain.User{}, domain.ErrUserNotFound
	}
	return s.GetUserByID(ctx, userID)
}

// GetUserByID возвращает пользователя.
func (s *txStorage) GetUserByID(ctx context.Context, userID string) (domain.User, error) {
	var u domain.User
	err := s.tx.QueryRow(ctx, `
		SELECT user_id, username, team_name, is_active
		FROM users
		WHERE user_id=$1
	`, userID).Scan(&u.ID, &u.Username, &u.TeamName, &u.IsActive)
	if err == pgx.ErrNoRows {
		return domain.User{}, domain.ErrUserNotFound
	}
	return u, err
}

// ListActiveTeamMembers возвращает активных участников команды.
func (s *txStorage) ListActiveTeamMembers(ctx context.Context, teamName string, exclude []string) ([]domain.User, error) {
	var params []any
	params = append(params, teamName)
	var where strings.Builder
	where.WriteString("team_name=$1 AND is_active=TRUE")
	if len(exclude) > 0 {
		ph := make([]string, len(exclude))
		for i, id := range exclude {
			params = append(params, id)
			ph[i] = fmt.Sprintf("$%d", i+2)
		}
		where.WriteString(" AND user_id NOT IN (" + strings.Join(ph, ",") + ")")
	}
	query := fmt.Sprintf(`SELECT user_id, username, team_name, is_active FROM users WHERE %s`, where.String())
	rows, err := s.tx.Query(ctx, query, params...)
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
func (s *txStorage) DeactivateUsers(ctx context.Context, userIDs []string) ([]domain.User, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	if _, err := s.tx.Exec(ctx, `
		UPDATE users SET is_active=FALSE, updated_at=NOW()
		WHERE user_id = ANY($1)
	`, userIDs); err != nil {
		return nil, err
	}
	rows, err := s.tx.Query(ctx, `
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

// CreatePullRequest создаёт PR и возвращает его.
func (s *txStorage) CreatePullRequest(ctx context.Context, pr domain.PullRequest, reviewers []string) (domain.PullRequest, error) {
	var exists bool
	if err := s.tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pull_requests WHERE pull_request_id=$1)`, pr.ID).Scan(&exists); err != nil {
		return domain.PullRequest{}, err
	}
	if exists {
		return domain.PullRequest{}, domain.ErrPRExists
	}
	if _, err := s.tx.Exec(ctx, `
		INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, created_at)
		VALUES ($1,$2,$3,$4,NOW())
	`, pr.ID, pr.Name, pr.AuthorID, string(pr.Status)); err != nil {
		return domain.PullRequest{}, err
	}
	if len(reviewers) > 0 {
		if err := insertReviewers(ctx, s.tx, pr.ID, reviewers, "AUTO_ASSIGN"); err != nil {
			return domain.PullRequest{}, err
		}
	}
	return s.GetPullRequest(ctx, pr.ID)
}

// GetPullRequest возвращает полный PR.
func (s *txStorage) GetPullRequest(ctx context.Context, prID string) (domain.PullRequest, error) {
	var pr domain.PullRequest
	var mergedAt *time.Time
	err := s.tx.QueryRow(ctx, `
		SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at
		FROM pull_requests WHERE pull_request_id=$1
	`, prID).Scan(&pr.ID, &pr.Name, &pr.AuthorID, &pr.Status, &pr.CreatedAt, &mergedAt)
	if err == pgx.ErrNoRows {
		return domain.PullRequest{}, domain.ErrPRNotFound
	}
	if err != nil {
		return domain.PullRequest{}, err
	}
	pr.MergedAt = mergedAt
	rows, err := s.tx.Query(ctx, `
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

// UpdatePRStatus устанавливает статус и merged_at.
func (s *txStorage) UpdatePRStatus(ctx context.Context, prID string, status domain.PRStatus) (domain.PullRequest, error) {
	var current domain.PRStatus
	var mergedAt *time.Time
	if err := s.tx.QueryRow(ctx, `
		SELECT status, merged_at FROM pull_requests WHERE pull_request_id=$1
	`, prID).Scan(&current, &mergedAt); err != nil {
		if err == pgx.ErrNoRows {
			return domain.PullRequest{}, domain.ErrPRNotFound
		}
		return domain.PullRequest{}, err
	}
	if current == status && status == domain.PRStatusMerged && mergedAt != nil {
		return s.GetPullRequest(ctx, prID)
	}
	if status == domain.PRStatusMerged {
		if _, err := s.tx.Exec(ctx, `
			UPDATE pull_requests SET status=$2, merged_at=COALESCE(merged_at, NOW())
			WHERE pull_request_id=$1
		`, prID, string(status)); err != nil {
			return domain.PullRequest{}, err
		}
	}
	return s.GetPullRequest(ctx, prID)
}

// ReplaceReviewer меняет одного ревьювера на другого.
func (s *txStorage) ReplaceReviewer(ctx context.Context, prID, oldReviewer, newReviewer, source string) (domain.PullRequest, string, error) {
	var status domain.PRStatus
	if err := s.tx.QueryRow(ctx, `SELECT status FROM pull_requests WHERE pull_request_id=$1`, prID).Scan(&status); err != nil {
		if err == pgx.ErrNoRows {
			return domain.PullRequest{}, "", domain.ErrPRNotFound
		}
		return domain.PullRequest{}, "", err
	}
	if status == domain.PRStatusMerged {
		return domain.PullRequest{}, "", domain.ErrPRMerged
	}
	var assigned bool
	if err := s.tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM pull_request_reviewers WHERE pull_request_id=$1 AND reviewer_id=$2
		)`, prID, oldReviewer).Scan(&assigned); err != nil {
		return domain.PullRequest{}, "", err
	}
	if !assigned {
		return domain.PullRequest{}, "", domain.ErrReviewerAbsent
	}
	if err := removeReviewer(ctx, s.tx, prID, oldReviewer, source); err != nil {
		return domain.PullRequest{}, "", err
	}
	if newReviewer != "" {
		if err := insertReviewers(ctx, s.tx, prID, []string{newReviewer}, source); err != nil {
			return domain.PullRequest{}, "", err
		}
	}
	pr, err := s.GetPullRequest(ctx, prID)
	return pr, newReviewer, err
}

// ListReviewAssignments возвращает PR'ы для ревьювера.
func (s *txStorage) ListReviewAssignments(ctx context.Context, userID string) ([]domain.PullRequestShort, error) {
	rows, err := s.tx.Query(ctx, `
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

// ListOpenPRsByReviewer возвращает открытые PR по ревьюверам.
func (s *txStorage) ListOpenPRsByReviewer(ctx context.Context, reviewerIDs []string) (map[string][]string, error) {
	if len(reviewerIDs) == 0 {
		return map[string][]string{}, nil
	}
	rows, err := s.tx.Query(ctx, `
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
func (s *txStorage) FetchAssignmentStats(ctx context.Context) (domain.AssignmentStats, error) {
	rows, err := s.tx.Query(ctx, `
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
	rows2, err := s.tx.Query(ctx, `
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
