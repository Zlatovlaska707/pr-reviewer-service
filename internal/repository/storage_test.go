package repository

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	pgxmock "github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"

	"pr-reviewer-service_Avito/internal/domain"
)

type stubNower struct {
	now time.Time
}

func (s stubNower) Now() time.Time {
	return s.now
}

func newMockStorage(t *testing.T) (*Storage, pgxmock.PgxPoolIface, stubNower) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, mock.ExpectationsWereMet())
		mock.Close()
	})
	nower := stubNower{now: time.Unix(0, 0)}
	return New(mock, nower), mock, nower
}

func TestStorageCreateTeamInsertsMembers(t *testing.T) {
	storage, mock, n := newMockStorage(t)
	ctx := context.Background()

	mock.ExpectBeginTx(pgx.TxOptions{})
	mock.ExpectQuery(`SELECT EXISTS`).WithArgs("backend").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectExec(`INSERT INTO teams`).WithArgs("backend").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec(`INSERT INTO users`).
		WithArgs("u1", "Alice", "backend", true, n.now, n.now).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec(`INSERT INTO users`).
		WithArgs("u2", "Bob", "backend", true, n.now, n.now).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()

	rows := pgxmock.NewRows([]string{"user_id", "username", "team_name", "is_active"}).
		AddRow("u1", "Alice", "backend", true).
		AddRow("u2", "Bob", "backend", true)
	mock.ExpectQuery(`SELECT u\.user_id`).WithArgs("backend").WillReturnRows(rows)

	team := domain.Team{
		Name: "backend",
		Members: []domain.User{
			{ID: "u1", Username: "Alice", IsActive: true},
			{ID: "u2", Username: "Bob", IsActive: true},
		},
	}

	created, err := storage.CreateTeam(ctx, team)
	require.NoError(t, err)
	require.Equal(t, "backend", created.Name)
	require.Len(t, created.Members, 2)
}

func TestStorageGetTeamNotFound(t *testing.T) {
	storage, mock, _ := newMockStorage(t)
	ctx := context.Background()

	mock.ExpectQuery(`SELECT u\.user_id`).WithArgs("ghost").
		WillReturnRows(pgxmock.NewRows([]string{"user_id", "username", "team_name", "is_active"}))
	mock.ExpectQuery(`SELECT EXISTS`).WithArgs("ghost").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))

	_, err := storage.GetTeam(ctx, "ghost")
	require.ErrorIs(t, err, domain.ErrTeamNotFound)
}

func TestStorageSetUserActivityUpdatesAndReturnsUser(t *testing.T) {
	storage, mock, n := newMockStorage(t)
	ctx := context.Background()

	mock.ExpectExec(`UPDATE users SET`).WithArgs(false, n.now, "u1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectQuery(`SELECT user_id`).WithArgs("u1").
		WillReturnRows(pgxmock.NewRows([]string{"user_id", "username", "team_name", "is_active"}).
			AddRow("u1", "Alice", "backend", false))

	user, err := storage.SetUserActivity(ctx, "u1", false)
	require.NoError(t, err)
	require.Equal(t, "u1", user.ID)
	require.False(t, user.IsActive)
}

func TestStorageGetUserByIDNotFound(t *testing.T) {
	storage, mock, _ := newMockStorage(t)
	ctx := context.Background()

	mock.ExpectQuery(`SELECT user_id`).WithArgs("missing").
		WillReturnError(pgx.ErrNoRows)

	_, err := storage.GetUserByID(ctx, "missing")
	require.ErrorIs(t, err, domain.ErrUserNotFound)
}

func TestStorageCreatePullRequestPersistsReviewers(t *testing.T) {
	storage, mock, _ := newMockStorage(t)
	ctx := context.Background()

	mock.ExpectBeginTx(pgx.TxOptions{})
	mock.ExpectQuery(`SELECT EXISTS`).WithArgs("pr-1").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectExec(`INSERT INTO pull_requests`).WithArgs("pr-1", "Feature", "u1", string(domain.PRStatusOpen)).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	batch := mock.ExpectBatch()
	batch.ExpectExec(`INSERT INTO pull_request_reviewers`).WithArgs("pr-1", "u2").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	batch.ExpectExec(`INSERT INTO review_assignment_events`).WithArgs("pr-1", "u2", "AUTO_ASSIGN").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()

	now := time.Now()
	mock.ExpectQuery(`SELECT pull_request_id`).WithArgs("pr-1").
		WillReturnRows(pgxmock.NewRows([]string{"pull_request_id", "pull_request_name", "author_id", "status", "created_at", "merged_at"}).
			AddRow("pr-1", "Feature", "u1", domain.PRStatusOpen, now, nil))
	mock.ExpectQuery(`SELECT reviewer_id`).WithArgs("pr-1").
		WillReturnRows(pgxmock.NewRows([]string{"reviewer_id"}).AddRow("u2"))

	pr := domain.PullRequest{ID: "pr-1", Name: "Feature", AuthorID: "u1", Status: domain.PRStatusOpen}
	created, err := storage.CreatePullRequest(ctx, pr, []string{"u2"})
	require.NoError(t, err)
	require.Equal(t, []string{"u2"}, created.AssignedReviewers)
}

func TestStorageGetPullRequestNotFound(t *testing.T) {
	storage, mock, _ := newMockStorage(t)
	ctx := context.Background()

	mock.ExpectQuery(`SELECT pull_request_id`).WithArgs("missing").
		WillReturnError(pgx.ErrNoRows)

	_, err := storage.GetPullRequest(ctx, "missing")
	require.ErrorIs(t, err, domain.ErrPRNotFound)
}

func TestStorageListReviewAssignments(t *testing.T) {
	storage, mock, _ := newMockStorage(t)
	ctx := context.Background()

	rows := pgxmock.NewRows([]string{"pull_request_id", "pull_request_name", "author_id", "status"}).
		AddRow("pr-1", "Feature", "u1", domain.PRStatusOpen)
	mock.ExpectQuery(`SELECT p\.pull_request_id`).WithArgs("u2").WillReturnRows(rows)

	assignments, err := storage.ListReviewAssignments(ctx, "u2")
	require.NoError(t, err)
	require.Len(t, assignments, 1)
	require.Equal(t, "pr-1", assignments[0].ID)
}

func TestStorageListActiveTeamMembersExcludes(t *testing.T) {
	storage, mock, _ := newMockStorage(t)
	ctx := context.Background()

	rows := pgxmock.NewRows([]string{"user_id", "username", "team_name", "is_active"}).
		AddRow("u3", "Charlie", "backend", true)
	mock.ExpectQuery(`SELECT user_id, username, team_name, is_active FROM users WHERE team_name=\$1 AND is_active=TRUE AND user_id NOT IN`).
		WithArgs("backend", "u1", "u2").WillReturnRows(rows)

	users, err := storage.ListActiveTeamMembers(ctx, "backend", []string{"u1", "u2"})
	require.NoError(t, err)
	require.Equal(t, []string{"u3"}, []string{users[0].ID})
}

func TestStorageDeactivateUsers(t *testing.T) {
	storage, mock, _ := newMockStorage(t)
	ctx := context.Background()

	mock.ExpectBeginTx(pgx.TxOptions{})
	mock.ExpectExec(`UPDATE users SET is_active=FALSE`).WithArgs(pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 2))
	mock.ExpectCommit()

	rows := pgxmock.NewRows([]string{"user_id", "username", "team_name", "is_active"}).
		AddRow("u1", "Alice", "backend", false).
		AddRow("u2", "Bob", "backend", false)
	mock.ExpectQuery(`SELECT user_id, username, team_name, is_active FROM users WHERE user_id = ANY`).
		WithArgs(pgxmock.AnyArg()).WillReturnRows(rows)

	users, err := storage.DeactivateUsers(ctx, []string{"u1", "u2"})
	require.NoError(t, err)
	require.Len(t, users, 2)
	require.False(t, users[0].IsActive)
}

func TestStorageListOpenPRsByReviewer(t *testing.T) {
	storage, mock, _ := newMockStorage(t)
	ctx := context.Background()

	rows := pgxmock.NewRows([]string{"reviewer_id", "pull_request_id"}).
		AddRow("u2", "pr-1").
		AddRow("u2", "pr-2")
	mock.ExpectQuery(`SELECT r\.reviewer_id`).WithArgs(pgxmock.AnyArg()).WillReturnRows(rows)

	result, err := storage.ListOpenPRsByReviewer(ctx, []string{"u2"})
	require.NoError(t, err)
	require.Equal(t, []string{"pr-1", "pr-2"}, result["u2"])
}

func TestStorageFetchAssignmentStats(t *testing.T) {
	storage, mock, _ := newMockStorage(t)
	ctx := context.Background()

	userRows := pgxmock.NewRows([]string{"user_id", "username", "team_name", "assigned_total", "active_pull_requests"}).
		AddRow("u1", "Alice", "backend", int64(3), int64(1))
	mock.ExpectQuery(`SELECT u\.user_id`).WillReturnRows(userRows)

	prRows := pgxmock.NewRows([]string{"pull_request_id", "reviewer_count"}).
		AddRow("pr-1", int64(2))
	mock.ExpectQuery(`SELECT p\.pull_request_id`).WillReturnRows(prRows)

	stats, err := storage.FetchAssignmentStats(ctx)
	require.NoError(t, err)
	require.Len(t, stats.PerUser, 1)
	require.Len(t, stats.PerPR, 1)
	require.Equal(t, "u1", stats.PerUser[0].UserID)
	require.Equal(t, int64(2), stats.PerPR[0].ReviewerCount)
}

func TestStorageUpdatePRStatusMerges(t *testing.T) {
	storage, mock, _ := newMockStorage(t)
	ctx := context.Background()

	mock.ExpectBeginTx(pgx.TxOptions{})
	mock.ExpectQuery(`SELECT status, merged_at`).WithArgs("pr-1").
		WillReturnRows(pgxmock.NewRows([]string{"status", "merged_at"}).AddRow(domain.PRStatusOpen, nil))
	mock.ExpectExec(`UPDATE pull_requests SET status=`).WithArgs("pr-1", string(domain.PRStatusMerged)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectCommit()

	now := time.Now()
	mergedAt := now
	mock.ExpectQuery(`SELECT pull_request_id`).WithArgs("pr-1").
		WillReturnRows(pgxmock.NewRows([]string{"pull_request_id", "pull_request_name", "author_id", "status", "created_at", "merged_at"}).
			AddRow("pr-1", "Feature", "u1", domain.PRStatusMerged, now, &mergedAt))
	mock.ExpectQuery(`SELECT reviewer_id`).WithArgs("pr-1").
		WillReturnRows(pgxmock.NewRows([]string{"reviewer_id"}))

	pr, err := storage.UpdatePRStatus(ctx, "pr-1", domain.PRStatusMerged)
	require.NoError(t, err)
	require.Equal(t, domain.PRStatusMerged, pr.Status)
	require.NotNil(t, pr.MergedAt)
}

func TestStorageReplaceReviewer(t *testing.T) {
	storage, mock, _ := newMockStorage(t)
	ctx := context.Background()

	mock.ExpectBeginTx(pgx.TxOptions{})
	mock.ExpectQuery(`SELECT status FROM pull_requests`).WithArgs("pr-1").
		WillReturnRows(pgxmock.NewRows([]string{"status"}).AddRow(domain.PRStatusOpen))
	mock.ExpectQuery(`SELECT EXISTS`).WithArgs("pr-1", "old").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))

	removeBatch := mock.ExpectBatch()
	removeBatch.ExpectExec(`DELETE FROM pull_request_reviewers`).WithArgs("pr-1", "old").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	removeBatch.ExpectExec(`INSERT INTO review_assignment_events`).WithArgs("pr-1", "old", "MANUAL").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	addBatch := mock.ExpectBatch()
	addBatch.ExpectExec(`INSERT INTO pull_request_reviewers`).WithArgs("pr-1", "new").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	addBatch.ExpectExec(`INSERT INTO review_assignment_events`).WithArgs("pr-1", "new", "MANUAL").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	mock.ExpectCommit()

	now := time.Now()
	mock.ExpectQuery(`SELECT pull_request_id`).WithArgs("pr-1").
		WillReturnRows(pgxmock.NewRows([]string{"pull_request_id", "pull_request_name", "author_id", "status", "created_at", "merged_at"}).
			AddRow("pr-1", "Feature", "u1", domain.PRStatusOpen, now, nil))
	mock.ExpectQuery(`SELECT reviewer_id`).WithArgs("pr-1").
		WillReturnRows(pgxmock.NewRows([]string{"reviewer_id"}).AddRow("new"))

	pr, replacedBy, err := storage.ReplaceReviewer(ctx, "pr-1", "old", "new", "MANUAL")
	require.NoError(t, err)
	require.Equal(t, "new", replacedBy)
	require.Equal(t, []string{"new"}, pr.AssignedReviewers)
}

func TestStoragePingAndClose(t *testing.T) {
	storage, mock, _ := newMockStorage(t)
	ctx := context.Background()

	mock.ExpectPing().WillReturnError(nil)
	require.NoError(t, storage.Ping(ctx))

	mock.ExpectClose()
	storage.Close()
}
