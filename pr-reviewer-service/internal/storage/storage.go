package storage

import (
	"database/sql"
	"fmt"
	"log"
	"pr-reviewer-service/internal/models"

	_ "github.com/lib/pq"
)

// Storage - iface for db
type Storage interface {
	// Teams
	CreateTeam(teamName string) error
	GetTeam(teamName string) (*models.TeamResponse, error)
	TeamExists(teamName string) (bool, error)

	// Users
	CreateOrUpdateUser(user *models.User) error
	GetUser(userID string) (*models.User, error)
	SetUserActive(userID string, isActive bool) error
	GetActiveTeamMembers(teamName string, excludeUserID string) ([]models.User, error)

	// Pull Requests
	CreatePullRequest(pr *models.PullRequest) error
	GetPullRequest(prID string) (*models.PullRequest, error)
	MergePullRequest(prID string) error
	PRExists(prID string) (bool, error)

	// Reviewers
	AddReviewer(prID, userID string) error
	RemoveReviewer(prID, userID string) error
	GetReviewers(prID string) ([]string, error)
	IsReviewerAssigned(prID, userID string) (bool, error)
	GetPRsByReviewer(userID string) ([]models.PullRequestShort, error)
}

type PostgresStorage struct {
	db *sql.DB
}

// NewPostgresStorage create new connection
func NewPostgresStorage(connStr string) (*PostgresStorage, error) {
	
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	
	return &PostgresStorage{db: db}, nil
}

func (s *PostgresStorage) Close() error {
	return s.db.Close()
}

// TEAMS

func (s *PostgresStorage) CreateTeam(teamName string) error {
	query := "INSERT INTO teams (team_name) VALUES ($1)"
	
	_, err := s.db.Exec(query, teamName)
	if err != nil {
		return fmt.Errorf("failed to create team: %w", err)
	}
	
	return nil
}

func (s *PostgresStorage) TeamExists(teamName string) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM teams WHERE team_name = $1)"
	
	var exists bool
	err := s.db.QueryRow(query, teamName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check team existence: %w", err)
	}
	
	return exists, nil
}

// GetTeam return all team members
func (s *PostgresStorage) GetTeam(teamName string) (*models.TeamResponse, error) {
	exists, err := s.TeamExists(teamName)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("team not found")
	}
	
	query := `
		SELECT user_id, username, is_active 
		FROM users 
		WHERE team_name = $1
		ORDER BY username
	`
	
	rows, err := s.db.Query(query, teamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get team members: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Failed to close rows: %v", err)
		}
	}()
	
	var members []models.TeamMember
	for rows.Next() {
		var member models.TeamMember
		err := rows.Scan(&member.UserID, &member.Username, &member.IsActive)
		if err != nil {
			return nil, fmt.Errorf("failed to scan team member: %w", err)
		}
		members = append(members, member)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating team members: %w", err)
	}
	
	return &models.TeamResponse{
		TeamName: teamName,
		Members:  members,
	}, nil
}

// USERS

func (s *PostgresStorage) CreateOrUpdateUser(user *models.User) error {
	query := `
		INSERT INTO users (user_id, username, team_name, is_active)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) 
		DO UPDATE SET 
			username = EXCLUDED.username,
			team_name = EXCLUDED.team_name,
			is_active = EXCLUDED.is_active
	`
	
	_, err := s.db.Exec(query, user.UserID, user.Username, user.TeamName, user.IsActive)
	if err != nil {
		return fmt.Errorf("failed to create or update user: %w", err)
	}
	
	return nil
}

func (s *PostgresStorage) GetUser(userID string) (*models.User, error) {
	query := `
		SELECT user_id, username, team_name, is_active
		FROM users
		WHERE user_id = $1
	`
	
	var user models.User
	err := s.db.QueryRow(query, userID).Scan(
		&user.UserID,
		&user.Username,
		&user.TeamName,
		&user.IsActive,
	)
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	
	return &user, nil
}

func (s *PostgresStorage) SetUserActive(userID string, isActive bool) error {
	query := "UPDATE users SET is_active = $1 WHERE user_id = $2"
	
	result, err := s.db.Exec(query, isActive, userID)
	if err != nil {
		return fmt.Errorf("failed to set user active: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}
	
	return nil
}

func (s *PostgresStorage) GetActiveTeamMembers(teamName string, excludeUserID string) ([]models.User, error) {
	query := `
		SELECT user_id, username, team_name, is_active
		FROM users
		WHERE team_name = $1 
		AND is_active = true 
		AND user_id != $2
		ORDER BY user_id
	`
	
	rows, err := s.db.Query(query, teamName, excludeUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active team members: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Failed to close rows: %v", err)
		}
	}()
	
	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}
	
	return users, nil
}

// PULL REQUESTS

func (s *PostgresStorage) CreatePullRequest(pr *models.PullRequest) error {
	query := `
		INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	
	_, err := s.db.Exec(query, 
		pr.PullRequestID, 
		pr.PullRequestName, 
		pr.AuthorID, 
		pr.Status,
		pr.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create pull request: %w", err)
	}
	
	return nil
}

func (s *PostgresStorage) PRExists(prID string) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM pull_requests WHERE pull_request_id = $1)"
	
	var exists bool
	err := s.db.QueryRow(query, prID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check PR existence: %w", err)
	}
	
	return exists, nil
}

func (s *PostgresStorage) GetPullRequest(prID string) (*models.PullRequest, error) {
	query := `
		SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at
		FROM pull_requests
		WHERE pull_request_id = $1
	`
	
	var pr models.PullRequest
	err := s.db.QueryRow(query, prID).Scan(
		&pr.PullRequestID,
		&pr.PullRequestName,
		&pr.AuthorID,
		&pr.Status,
		&pr.CreatedAt,
		&pr.MergedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("pull request not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get pull request: %w", err)
	}
	
	reviewers, err := s.GetReviewers(prID)
	if err != nil {
		return nil, err
	}
	pr.AssignedReviewers = reviewers
	
	return &pr, nil
}

// MergePullRequest marks PR as MERGED (idempotent operation)
func (s *PostgresStorage) MergePullRequest(prID string) error {
	query := `
		UPDATE pull_requests 
		SET status = 'MERGED', merged_at = CURRENT_TIMESTAMP
		WHERE pull_request_id = $1 AND status = 'OPEN'
	`
	
	result, err := s.db.Exec(query, prID)
	if err != nil {
		return fmt.Errorf("failed to merge pull request: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		exists, err := s.PRExists(prID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("pull request not found")
		}
	}
	
	return nil
}

// REVIEWERS

func (s *PostgresStorage) AddReviewer(prID, userID string) error {
	query := `
		INSERT INTO pr_reviewers (pull_request_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`
	
	_, err := s.db.Exec(query, prID, userID)
	if err != nil {
		return fmt.Errorf("failed to add reviewer: %w", err)
	}
	
	return nil
}

func (s *PostgresStorage) RemoveReviewer(prID, userID string) error {
	query := "DELETE FROM pr_reviewers WHERE pull_request_id = $1 AND user_id = $2"
	
	_, err := s.db.Exec(query, prID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove reviewer: %w", err)
	}
	
	return nil
}

func (s *PostgresStorage) GetReviewers(prID string) ([]string, error) {
	query := `
		SELECT user_id 
		FROM pr_reviewers 
		WHERE pull_request_id = $1
		ORDER BY user_id
	`
	
	rows, err := s.db.Query(query, prID)
	if err != nil {
		return nil, fmt.Errorf("failed to get reviewers: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Failed to close rows: %v", err)
		}
	}()
	
	var reviewers []string
	for rows.Next() {
		var userID string
		err := rows.Scan(&userID)
		if err != nil {
			return nil, fmt.Errorf("failed to scan reviewer: %w", err)
		}
		reviewers = append(reviewers, userID)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating reviewers: %w", err)
	}
	
	return reviewers, nil
}

// IsReviewerAssigned checks if user is assigned as reviewer for PR
func (s *PostgresStorage) IsReviewerAssigned(prID, userID string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM pr_reviewers 
			WHERE pull_request_id = $1 AND user_id = $2
		)
	`
	
	var assigned bool
	err := s.db.QueryRow(query, prID, userID).Scan(&assigned)
	if err != nil {
		return false, fmt.Errorf("failed to check reviewer assignment: %w", err)
	}
	
	return assigned, nil
}

// GetPRsByReviewer returns all PRs where user is reviewer
func (s *PostgresStorage) GetPRsByReviewer(userID string) ([]models.PullRequestShort, error) {
	query := `
		SELECT pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status
		FROM pull_requests pr
		INNER JOIN pr_reviewers r ON pr.pull_request_id = r.pull_request_id
		WHERE r.user_id = $1
		ORDER BY pr.created_at DESC
	`
	
	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get PRs by reviewer: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Failed to close rows: %v", err)
		}
	}()
	
	var prs []models.PullRequestShort
	for rows.Next() {
		var pr models.PullRequestShort
		err := rows.Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status)
		if err != nil {
			return nil, fmt.Errorf("failed to scan PR: %w", err)
		}
		prs = append(prs, pr)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating PRs: %w", err)
	}
	
	return prs, nil
}
