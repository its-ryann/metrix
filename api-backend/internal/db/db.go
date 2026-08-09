package db

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Name         string    `json:"name"`
	CreatedAt    time.Time `json:"created_at"`
}

type PlatformAccount struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Platform    string    `json:"platform"`
	DisplayName string    `json:"display_name"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type MetricsSummary struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	TotalReach     int       `json:"total_reach"`
	AvgEngagement  float64   `json:"avg_engagement"`
	FollowerGrowth int       `json:"follower_growth"`
	RecordedAt     time.Time `json:"recorded_at"`
}

type TimeSeriesPoint struct {
	Date  string `json:"date"`
	Value int    `json:"value"`
}

type ContentItem struct {
	Title      string  `json:"title"`
	Platform   string  `json:"platform"`
	Engagement float64 `json:"engagement"`
	Reach      int     `json:"reach"`
}

func InitDB(ctx context.Context, connStr string) error {
	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return fmt.Errorf("unable to parse DATABASE_URL: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return fmt.Errorf("unable to create db pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("unable to ping database: %w", err)
	}

	Pool = pool
	log.Println("Database connection pool established successfully.")
	return nil
}

func GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `SELECT id, email, password_hash, name, created_at FROM users WHERE email = $1`
	row := Pool.QueryRow(ctx, query, email)

	var u User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func GetUserByID(ctx context.Context, id string) (*User, error) {
	query := `SELECT id, email, password_hash, name, created_at FROM users WHERE id = $1`
	row := Pool.QueryRow(ctx, query, id)

	var u User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func CreateUser(ctx context.Context, email, passwordHash, name string) (*User, error) {
	query := `
		INSERT INTO users (email, password_hash, name)
		VALUES ($1, $2, $3)
		RETURNING id, email, password_hash, name, created_at
	`
	var u User
	err := Pool.QueryRow(ctx, query, email, passwordHash, name).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert user: %w", err)
	}

	// Seed default platform accounts for user
	platformAccounts := []struct {
		platform    string
		displayName string
	}{
		{"youtube", name + " Channel"},
		{"instagram", "@" + email[:indexOrLength(email, "@")] + "_official"},
		{"tiktok", "@" + email[:indexOrLength(email, "@")]},
	}

	for _, pa := range platformAccounts {
		_, _ = Pool.Exec(ctx, `
			INSERT INTO platform_accounts (user_id, platform, display_name, status)
			VALUES ($1, $2, $3, 'connected')
		`, u.ID, pa.platform, pa.displayName)
	}

	return &u, nil
}

func indexOrLength(s, substr string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == '@' {
			return i
		}
	}
	return len(s)
}

func GetSummaryForUser(ctx context.Context, userID string) (*MetricsSummary, error) {
	query := `
		SELECT id, user_id, total_reach, avg_engagement, follower_growth, recorded_at
		FROM metrics_summary
		WHERE user_id = $1
		ORDER BY recorded_at DESC
		LIMIT 1
	`
	var ms MetricsSummary
	err := Pool.QueryRow(ctx, query, userID).Scan(
		&ms.ID, &ms.UserID, &ms.TotalReach, &ms.AvgEngagement, &ms.FollowerGrowth, &ms.RecordedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Return default 0 summary if pipeline hasn't ingested yet
			return &MetricsSummary{
				UserID:         userID,
				TotalReach:     0,
				AvgEngagement:  0.0,
				FollowerGrowth: 0,
				RecordedAt:     time.Now(),
			}, nil
		}
		return nil, err
	}
	return &ms, nil
}

func GetTimeSeriesForUser(ctx context.Context, userID string, platform string) ([]TimeSeriesPoint, error) {
	var query string
	var rows pgx.Rows
	var err error

	if platform != "" && platform != "all" {
		query = `
			SELECT t.metric_date, SUM(t.value) as value
			FROM metrics_timeseries t
			JOIN platform_accounts pa ON pa.user_id = t.user_id AND pa.platform = t.platform AND pa.status = 'connected'
			WHERE t.user_id = $1 AND t.platform = $2
			GROUP BY t.metric_date
			ORDER BY t.metric_date ASC
			LIMIT 14
		`
		rows, err = Pool.Query(ctx, query, userID, platform)
	} else {
		query = `
			SELECT t.metric_date, SUM(t.value) as value
			FROM metrics_timeseries t
			JOIN platform_accounts pa ON pa.user_id = t.user_id AND pa.platform = t.platform AND pa.status = 'connected'
			WHERE t.user_id = $1
			GROUP BY t.metric_date
			ORDER BY t.metric_date ASC
			LIMIT 14
		`
		rows, err = Pool.Query(ctx, query, userID)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []TimeSeriesPoint
	for rows.Next() {
		var metricDate time.Time
		var val int
		if err := rows.Scan(&metricDate, &val); err != nil {
			return nil, err
		}
		points = append(points, TimeSeriesPoint{
			Date:  metricDate.Format("2006-01-02"),
			Value: val,
		})
	}

	if len(points) == 0 {
		// Fallback to empty list or last 7 days placeholder dates if no data yet
		now := time.Now()
		for i := 6; i >= 0; i-- {
			points = append(points, TimeSeriesPoint{
				Date:  now.AddDate(0, 0, -i).Format("2006-01-02"),
				Value: 0,
			})
		}
	}

	return points, nil
}

func GetPlatformAccountsForUser(ctx context.Context, userID string) ([]PlatformAccount, error) {
	query := `
		SELECT id, user_id, platform, display_name, status, created_at
		FROM platform_accounts
		WHERE user_id = $1
		ORDER BY created_at ASC
	`
	rows, err := Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []PlatformAccount
	for rows.Next() {
		var pa PlatformAccount
		if err := rows.Scan(&pa.ID, &pa.UserID, &pa.Platform, &pa.DisplayName, &pa.Status, &pa.CreatedAt); err != nil {
			return nil, err
		}
		accounts = append(accounts, pa)
	}
	return accounts, nil
}

func GetPlatformAccountForUser(ctx context.Context, userID, platform string) (*PlatformAccount, error) {
	query := `
		SELECT id, user_id, platform, display_name, status, created_at
		FROM platform_accounts
		WHERE user_id = $1 AND platform = $2
		ORDER BY created_at ASC
		LIMIT 1
	`
	var pa PlatformAccount
	err := Pool.QueryRow(ctx, query, userID, platform).Scan(
		&pa.ID, &pa.UserID, &pa.Platform, &pa.DisplayName, &pa.Status, &pa.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &pa, nil
}

func CreatePlatformAccount(ctx context.Context, userID, platform, displayName string) (*PlatformAccount, error) {
	query := `
		INSERT INTO platform_accounts (user_id, platform, display_name, status)
		VALUES ($1, $2, $3, 'connected')
		RETURNING id, user_id, platform, display_name, status, created_at
	`
	var pa PlatformAccount
	err := Pool.QueryRow(ctx, query, userID, platform, displayName).Scan(
		&pa.ID, &pa.UserID, &pa.Platform, &pa.DisplayName, &pa.Status, &pa.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create platform account: %w", err)
	}
	return &pa, nil
}

func UpdatePlatformAccountStatus(ctx context.Context, accountID, userID, status string) (bool, error) {
	tag, err := Pool.Exec(ctx, `
		UPDATE platform_accounts
		SET status = $1
		WHERE id = $2 AND user_id = $3
	`, status, accountID, userID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// PlatformStats holds per-account activity used to enrich the platforms view.
type PlatformStats struct {
	Followers    int
	LastSyncedAt *time.Time
}

func GetPlatformStats(ctx context.Context, userID, platform string) (PlatformStats, error) {
	var stats PlatformStats
	err := Pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(value), 0), MAX(metric_date)
		FROM metrics_timeseries
		WHERE user_id = $1 AND platform = $2
		  AND metric_date >= CURRENT_DATE - 14
	`, userID, platform).Scan(&stats.Followers, &stats.LastSyncedAt)
	if err != nil {
		return stats, err
	}
	return stats, nil
}

func GetConnectedPlatforms(ctx context.Context, userID string) ([]string, error) {
	rows, err := Pool.Query(ctx, `
		SELECT platform FROM platform_accounts
		WHERE user_id = $1 AND status = 'connected'
		ORDER BY created_at ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var platforms []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		platforms = append(platforms, p)
	}
	return platforms, nil
}

// SeedMetricsForPlatform backfills 14 days of timeseries data so a freshly
// connected account shows history immediately instead of an empty chart.
func SeedMetricsForPlatform(ctx context.Context, userID, platform string) error {
	now := time.Now()
	base := 4000
	switch platform {
	case "instagram":
		base = 2500
	case "tiktok":
		base = 6000
	}
	for i := 13; i >= 0; i-- {
		date := now.AddDate(0, 0, -i).Format("2006-01-02")
		value := base + int(float64(base)*0.5*((rand.Float64()*2)-1))
		if _, err := Pool.Exec(ctx, `
			INSERT INTO metrics_timeseries (user_id, platform, metric_date, value)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT DO NOTHING
		`, userID, platform, date, value); err != nil {
			return err
		}
	}
	return nil
}

func UpdateUserName(ctx context.Context, userID, name string) (*User, error) {
	query := `
		UPDATE users SET name = $1 WHERE id = $2
		RETURNING id, email, password_hash, name, created_at
	`
	var u User
	err := Pool.QueryRow(ctx, query, name, userID).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update user name: %w", err)
	}
	return &u, nil
}

func UpdateUserPassword(ctx context.Context, userID, passwordHash string) error {
	tag, err := Pool.Exec(ctx, `
		UPDATE users SET password_hash = $1 WHERE id = $2
	`, passwordHash, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("user not found")
	}
	return nil
}

func DeleteUser(ctx context.Context, userID string) error {
	tag, err := Pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("user not found")
	}
	return nil
}

func GetSummaryHistory(ctx context.Context, userID string, limit int) ([]MetricsSummary, error) {
	rows, err := Pool.Query(ctx, `
		SELECT id, user_id, total_reach, avg_engagement, follower_growth, recorded_at
		FROM metrics_summary
		WHERE user_id = $1
		ORDER BY recorded_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []MetricsSummary
	for rows.Next() {
		var ms MetricsSummary
		if err := rows.Scan(&ms.ID, &ms.UserID, &ms.TotalReach, &ms.AvgEngagement, &ms.FollowerGrowth, &ms.RecordedAt); err != nil {
			return nil, err
		}
		summaries = append(summaries, ms)
	}
	return summaries, nil
}
