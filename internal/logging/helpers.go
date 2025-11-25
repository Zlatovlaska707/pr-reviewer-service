package logging

import "context"

// WithLogRequestID добавляет request ID в контекст.
func WithLogRequestID(ctx context.Context, requestID string) context.Context {
	if c, ok := ctx.Value(key).(logCtx); ok {
		c.RequestID = requestID
		return context.WithValue(ctx, key, c)
	}
	return context.WithValue(ctx, key, logCtx{RequestID: requestID})
}

// WithLogRequestPath добавляет путь запроса в контекст.
func WithLogRequestPath(ctx context.Context, path string) context.Context {
	if c, ok := ctx.Value(key).(logCtx); ok {
		c.Path = path
		return context.WithValue(ctx, key, c)
	}
	return context.WithValue(ctx, key, logCtx{Path: path})
}

// WithLogRequestMethod добавляет метод запроса в контекст.
func WithLogRequestMethod(ctx context.Context, method string) context.Context {
	if c, ok := ctx.Value(key).(logCtx); ok {
		c.Method = method
		return context.WithValue(ctx, key, c)
	}
	return context.WithValue(ctx, key, logCtx{Method: method})
}

// WithLogRequestStatus добавляет статус ответа в контекст.
func WithLogRequestStatus(ctx context.Context, status int) context.Context {
	if c, ok := ctx.Value(key).(logCtx); ok {
		c.Status = status
		return context.WithValue(ctx, key, c)
	}
	return context.WithValue(ctx, key, logCtx{Status: status})
}

// WithLogRequestDuration добавляет длительность запроса в контекст.
func WithLogRequestDuration(ctx context.Context, duration string) context.Context {
	if c, ok := ctx.Value(key).(logCtx); ok {
		c.RequestDuration = duration
		return context.WithValue(ctx, key, c)
	}
	return context.WithValue(ctx, key, logCtx{RequestDuration: duration})
}

// WithLogTeamName добавляет имя команды в контекст.
func WithLogTeamName(ctx context.Context, teamName string) context.Context {
	if c, ok := ctx.Value(key).(logCtx); ok {
		c.TeamName = teamName
		return context.WithValue(ctx, key, c)
	}
	return context.WithValue(ctx, key, logCtx{TeamName: teamName})
}

// WithLogTeamMembersCount добавляет количество участников команды в контекст.
func WithLogTeamMembersCount(ctx context.Context, cnt int) context.Context {
	if c, ok := ctx.Value(key).(logCtx); ok {
		c.TeamMembersCount = cnt
		return context.WithValue(ctx, key, c)
	}
	return context.WithValue(ctx, key, logCtx{TeamMembersCount: cnt})
}

// WithLogUserID добавляет ID пользователя в контекст.
func WithLogUserID(ctx context.Context, userID string) context.Context {
	if c, ok := ctx.Value(key).(logCtx); ok {
		c.UserID = userID
		return context.WithValue(ctx, key, c)
	}
	return context.WithValue(ctx, key, logCtx{UserID: userID})
}

// WithLogAuthorID добавляет ID автора в контекст.
func WithLogAuthorID(ctx context.Context, authorID string) context.Context {
	if c, ok := ctx.Value(key).(logCtx); ok {
		c.AuthorID = authorID
		return context.WithValue(ctx, key, c)
	}
	return context.WithValue(ctx, key, logCtx{AuthorID: authorID})
}

// WithLogPullRequestID добавляет ID Pull Request в контекст.
func WithLogPullRequestID(ctx context.Context, prID string) context.Context {
	if c, ok := ctx.Value(key).(logCtx); ok {
		c.PullRequestID = prID
		return context.WithValue(ctx, key, c)
	}
	return context.WithValue(ctx, key, logCtx{PullRequestID: prID})
}
