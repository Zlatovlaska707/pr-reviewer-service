package service

import (
	"context"
	"sort"
	"testing"
	"time"

	trm "github.com/avito-tech/go-transaction-manager/trm/v2"
	"github.com/stretchr/testify/require"

	"pr-reviewer-service_Avito/internal/config"
	"pr-reviewer-service_Avito/internal/domain"
	randomizerpkg "pr-reviewer-service_Avito/internal/infrastructure/randomizer"
	"pr-reviewer-service_Avito/internal/repository"
)

func TestService_CreatePullRequest_AssignsTwoReviewers(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	author := domain.User{ID: "u1", TeamName: "backend", IsActive: true}
	candidates := []domain.User{
		{ID: "u2", TeamName: "backend", IsActive: true},
		{ID: "u3", TeamName: "backend", IsActive: true},
	}

	var capturedReviewers []string
	fake := &fakeRepo{
		getUserByIDFn: func(ctx context.Context, userID string) (domain.User, error) {
			return author, nil
		},
		listActiveTeamMembersFn: func(ctx context.Context, teamName string, exclude []string) ([]domain.User, error) {
			require.Equal(t, "backend", teamName)
			require.ElementsMatch(t, []string{"u1"}, exclude)
			return candidates, nil
		},
		createPullRequestFn: func(ctx context.Context, pr domain.PullRequest, reviewers []string) (domain.PullRequest, error) {
			capturedReviewers = append([]string{}, reviewers...)
			pr.AssignedReviewers = reviewers
			return pr, nil
		},
	}

	svc := New(fake, testConfig(), stubManager{}, stubRandomizer{})
	pr, err := svc.CreatePullRequest(ctx, "pr-1", "Add feature", "u1")
	require.NoError(t, err)
	require.Equal(t, "pr-1", pr.ID)
	require.Equal(t, "u1", pr.AuthorID)
	require.Len(t, capturedReviewers, 2)
	require.ElementsMatch(t, []string{"u2", "u3"}, capturedReviewers)
}

func TestService_ReassignReviewer_NoCandidate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fake := &fakeRepo{
		getPullRequestFn: func(ctx context.Context, prID string) (domain.PullRequest, error) {
			return domain.PullRequest{
				ID:                prID,
				Status:            domain.PRStatusOpen,
				AuthorID:          "author",
				AssignedReviewers: []string{"old"},
			}, nil
		},
		getUserByIDFn: func(ctx context.Context, userID string) (domain.User, error) {
			return domain.User{ID: userID, TeamName: "backend"}, nil
		},
		listActiveTeamMembersFn: func(ctx context.Context, teamName string, exclude []string) ([]domain.User, error) {
			return []domain.User{}, nil
		},
	}

	svc := New(fake, testConfig(), stubManager{}, stubRandomizer{})
	_, _, err := svc.ReassignReviewer(ctx, "pr-1", "old")
	require.ErrorIs(t, err, domain.ErrNoCandidate)
}

func TestService_MassDeactivate_ReassignsOpenPRs(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	replaced := make(map[string][]string)
	fake := &fakeRepo{
		getTeamFn: func(ctx context.Context, name string) (domain.Team, error) {
			return domain.Team{
				Name: name,
				Members: []domain.User{
					{ID: "u1", TeamName: name, IsActive: true},
					{ID: "u2", TeamName: name, IsActive: true},
					{ID: "u3", TeamName: name, IsActive: true},
				},
			}, nil
		},
		deactivateUsersFn: func(ctx context.Context, ids []string) ([]domain.User, error) {
			return []domain.User{{ID: "u2", TeamName: "backend", IsActive: false}}, nil
		},
		listOpenPRsByReviewerFn: func(ctx context.Context, reviewerIDs []string) (map[string][]string, error) {
			return map[string][]string{"u2": {"pr-1"}}, nil
		},
		getPullRequestFn: func(ctx context.Context, prID string) (domain.PullRequest, error) {
			return domain.PullRequest{
				ID:                prID,
				Status:            domain.PRStatusOpen,
				AuthorID:          "u1",
				AssignedReviewers: []string{"u2"},
			}, nil
		},
		listActiveTeamMembersFn: func(ctx context.Context, teamName string, exclude []string) ([]domain.User, error) {
			return []domain.User{{ID: "u3", TeamName: teamName, IsActive: true}}, nil
		},
		replaceReviewerFn: func(ctx context.Context, prID, oldReviewer, newReviewer, source string) (domain.PullRequest, string, error) {
			replaced[oldReviewer] = append(replaced[oldReviewer], prID)
			require.Equal(t, "TEAM_DEACTIVATION", source)
			return domain.PullRequest{}, newReviewer, nil
		},
	}

	svc := New(fake, testConfig(), stubManager{}, stubRandomizer{})
	result, err := svc.MassDeactivate(ctx, MassDeactivateInput{TeamName: "backend", UserIDs: []string{"u2"}})
	require.NoError(t, err)
	require.Len(t, result.Deactivated, 1)
	require.Equal(t, []string{"pr-1"}, result.Reassigned["u2"])
	require.Empty(t, result.Skipped)
	require.Equal(t, []string{"pr-1"}, replaced["u2"])
}

func TestPickRandomIDsRespectLimit(t *testing.T) {
	users := []domain.User{
		{ID: "u1"},
		{ID: "u2"},
		{ID: "u3"},
	}
	selected := pickRandomIDs(users, 2, stubRandomizer{})
	require.Equal(t, []string{"u1", "u2"}, selected)
}

func TestUniqueIDs(t *testing.T) {
	values := []string{"a", "b", "a", " ", "c"}
	result := uniqueIDs(values)
	sort.Strings(result)
	require.Equal(t, []string{"a", "b", "c"}, result)
}

func TestServiceCreateTeamCallsRepo(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	var called bool
	fake := &fakeRepo{
		createTeamFn: func(ctx context.Context, team domain.Team) (domain.Team, error) {
			called = true
			return team, nil
		},
		getTeamFn: func(ctx context.Context, name string) (domain.Team, error) {
			return domain.Team{Name: name}, nil
		},
	}
	svc := New(fake, testConfig(), stubManager{}, stubRandomizer{})
	team := domain.Team{
		Name: "backend",
		Members: []domain.User{
			{ID: "u1", Username: "Alice"},
		},
	}
	_, err := svc.CreateTeam(ctx, team)
	require.NoError(t, err)
	require.True(t, called)
}

func TestServiceGetTeamValidatesName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fake := &fakeRepo{
		getTeamFn: func(ctx context.Context, name string) (domain.Team, error) {
			return domain.Team{Name: name}, nil
		},
	}
	svc := New(fake, testConfig(), stubManager{}, stubRandomizer{})
	_, err := svc.GetTeam(ctx, "")
	require.Error(t, err)
	team, err := svc.GetTeam(ctx, "backend")
	require.NoError(t, err)
	require.Equal(t, "backend", team.Name)
}

func TestServiceSetUserActivity(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fake := &fakeRepo{
		setUserActivityFn: func(ctx context.Context, userID string, active bool) (domain.User, error) {
			require.Equal(t, "user-1", userID)
			require.False(t, active)
			return domain.User{ID: userID, IsActive: active}, nil
		},
	}
	svc := New(fake, testConfig(), stubManager{}, stubRandomizer{})
	user, err := svc.SetUserActivity(ctx, "user-1", false)
	require.NoError(t, err)
	require.False(t, user.IsActive)
}

func TestServiceMergePullRequest(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fake := &fakeRepo{
		updatePRStatusFn: func(ctx context.Context, prID string, status domain.PRStatus) (domain.PullRequest, error) {
			require.Equal(t, domain.PRStatusMerged, status)
			return domain.PullRequest{ID: prID, Status: status}, nil
		},
	}
	svc := New(fake, testConfig(), stubManager{}, stubRandomizer{})
	pr, err := svc.MergePullRequest(ctx, "pr-1")
	require.NoError(t, err)
	require.Equal(t, domain.PRStatusMerged, pr.Status)
}

func TestServiceListReviewAssignments(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	expected := []domain.PullRequestShort{{ID: "pr-1"}}
	fake := &fakeRepo{
		listReviewAssignmentsFn: func(ctx context.Context, userID string) ([]domain.PullRequestShort, error) {
			return expected, nil
		},
	}
	svc := New(fake, testConfig(), stubManager{}, stubRandomizer{})
	assignments, err := svc.ListReviewAssignments(ctx, "user-1")
	require.NoError(t, err)
	require.Equal(t, expected, assignments)
}

func TestServiceStats(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	stats := domain.AssignmentStats{
		PerUser: []domain.UserAssignmentStat{{UserID: "u1", Assigned: 2}},
	}
	fake := &fakeRepo{
		fetchAssignmentStatsFn: func(ctx context.Context) (domain.AssignmentStats, error) {
			return stats, nil
		},
	}
	svc := New(fake, testConfig(), stubManager{}, stubRandomizer{})
	result, err := svc.Stats(ctx)
	require.NoError(t, err)
	require.Equal(t, stats, result)
}

func TestServiceHealthCheckUsesRepo(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	var called bool
	fake := &fakeRepo{
		pingFn: func(ctx context.Context) error {
			called = true
			return nil
		},
	}
	svc := New(fake, testConfig(), stubManager{}, stubRandomizer{})
	require.NoError(t, svc.HealthCheck(ctx))
	require.True(t, called)
}

func TestServiceHealthCheckPropagatesError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fake := &fakeRepo{
		pingFn: func(ctx context.Context) error {
			return context.DeadlineExceeded
		},
	}
	svc := New(fake, testConfig(), stubManager{}, stubRandomizer{})
	require.Error(t, svc.HealthCheck(ctx))
}

func testConfig() config.Config {
	return config.Config{
		Timeouts: config.TimeoutConfig{
			Operation:     time.Second,
			LongOperation: 2 * time.Second,
		},
	}
}

type stubManager struct{}

func (stubManager) Do(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func (stubManager) DoWithSettings(ctx context.Context, _ trm.Settings, fn func(context.Context) error) error {
	return fn(ctx)
}

type stubRandomizer struct{}

func (stubRandomizer) Shuffle(n int, swap func(i, j int)) {}

var (
	_ trm.Manager              = stubManager{}
	_ randomizerpkg.Randomizer = stubRandomizer{}
)

// fakeRepo позволяет настраивать ответы для юнит-тестов.
type fakeRepo struct {
	createTeamFn            func(context.Context, domain.Team) (domain.Team, error)
	getTeamFn               func(context.Context, string) (domain.Team, error)
	setUserActivityFn       func(context.Context, string, bool) (domain.User, error)
	getUserByIDFn           func(context.Context, string) (domain.User, error)
	listActiveTeamMembersFn func(context.Context, string, []string) ([]domain.User, error)
	createPullRequestFn     func(context.Context, domain.PullRequest, []string) (domain.PullRequest, error)
	updatePRStatusFn        func(context.Context, string, domain.PRStatus) (domain.PullRequest, error)
	getPullRequestFn        func(context.Context, string) (domain.PullRequest, error)
	replaceReviewerFn       func(context.Context, string, string, string, string) (domain.PullRequest, string, error)
	listReviewAssignmentsFn func(context.Context, string) ([]domain.PullRequestShort, error)
	fetchAssignmentStatsFn  func(context.Context) (domain.AssignmentStats, error)
	deactivateUsersFn       func(context.Context, []string) ([]domain.User, error)
	listOpenPRsByReviewerFn func(context.Context, []string) (map[string][]string, error)
	pingFn                  func(context.Context) error
}

func (f *fakeRepo) CreateTeam(ctx context.Context, team domain.Team) (domain.Team, error) {
	if f.createTeamFn != nil {
		return f.createTeamFn(ctx, team)
	}
	return domain.Team{}, nil
}

func (f *fakeRepo) GetTeam(ctx context.Context, name string) (domain.Team, error) {
	if f.getTeamFn != nil {
		return f.getTeamFn(ctx, name)
	}
	return domain.Team{}, nil
}

func (f *fakeRepo) SetUserActivity(ctx context.Context, userID string, active bool) (domain.User, error) {
	if f.setUserActivityFn != nil {
		return f.setUserActivityFn(ctx, userID, active)
	}
	return domain.User{}, nil
}

func (f *fakeRepo) GetUserByID(ctx context.Context, userID string) (domain.User, error) {
	if f.getUserByIDFn != nil {
		return f.getUserByIDFn(ctx, userID)
	}
	return domain.User{}, nil
}

func (f *fakeRepo) ListActiveTeamMembers(ctx context.Context, teamName string, exclude []string) ([]domain.User, error) {
	if f.listActiveTeamMembersFn != nil {
		return f.listActiveTeamMembersFn(ctx, teamName, exclude)
	}
	return nil, nil
}

func (f *fakeRepo) CreatePullRequest(ctx context.Context, pr domain.PullRequest, reviewers []string) (domain.PullRequest, error) {
	if f.createPullRequestFn != nil {
		return f.createPullRequestFn(ctx, pr, reviewers)
	}
	return domain.PullRequest{}, nil
}

func (f *fakeRepo) UpdatePRStatus(ctx context.Context, prID string, status domain.PRStatus) (domain.PullRequest, error) {
	if f.updatePRStatusFn != nil {
		return f.updatePRStatusFn(ctx, prID, status)
	}
	return domain.PullRequest{}, nil
}

func (f *fakeRepo) GetPullRequest(ctx context.Context, prID string) (domain.PullRequest, error) {
	if f.getPullRequestFn != nil {
		return f.getPullRequestFn(ctx, prID)
	}
	return domain.PullRequest{}, nil
}

func (f *fakeRepo) ReplaceReviewer(ctx context.Context, prID, oldReviewer, newReviewer, source string) (domain.PullRequest, string, error) {
	if f.replaceReviewerFn != nil {
		return f.replaceReviewerFn(ctx, prID, oldReviewer, newReviewer, source)
	}
	return domain.PullRequest{}, "", nil
}

func (f *fakeRepo) ListReviewAssignments(ctx context.Context, userID string) ([]domain.PullRequestShort, error) {
	if f.listReviewAssignmentsFn != nil {
		return f.listReviewAssignmentsFn(ctx, userID)
	}
	return nil, nil
}

func (f *fakeRepo) FetchAssignmentStats(ctx context.Context) (domain.AssignmentStats, error) {
	if f.fetchAssignmentStatsFn != nil {
		return f.fetchAssignmentStatsFn(ctx)
	}
	return domain.AssignmentStats{}, nil
}

func (f *fakeRepo) DeactivateUsers(ctx context.Context, userIDs []string) ([]domain.User, error) {
	if f.deactivateUsersFn != nil {
		return f.deactivateUsersFn(ctx, userIDs)
	}
	return nil, nil
}

func (f *fakeRepo) ListOpenPRsByReviewer(ctx context.Context, reviewerIDs []string) (map[string][]string, error) {
	if f.listOpenPRsByReviewerFn != nil {
		return f.listOpenPRsByReviewerFn(ctx, reviewerIDs)
	}
	return map[string][]string{}, nil
}

func (f *fakeRepo) WithTransaction(ctx context.Context, fn func(repository.Repository) error) error {
	// В тестах просто вызываем функцию без реальной транзакции
	return fn(f)
}

func (f *fakeRepo) Ping(ctx context.Context) error {
	if f.pingFn != nil {
		return f.pingFn(ctx)
	}
	return nil
}
