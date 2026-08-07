package db

import (
	"context"
	"errors"
	"fmt"
	"log"
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
			SELECT metric_date, SUM(value) as value
			FROM metrics_timeseries
			WHERE user_id = $1 AND platform = $2
			GROUP BY metric_date
			ORDER BY metric_date ASC
			LIMIT 14
		`
		rows, err = Pool.Query(ctx, query, userID, platform)
	} else {
		query = `
			SELECT metric_date, SUM(value) as value
			FROM metrics_timeseries
			WHERE user_id = $1
			GROUP BY metric_date
			ORDER BY metric_date ASC
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
