package service

import (
	"math/rand"
	"time"
	"pr-reviewer-service/internal/models"
	"pr-reviewer-service/internal/storage"
)

// ServiceError - custom Error
type ServiceError struct {
	Code    string
	Message string
}

func (e *ServiceError) Error() string {
	return e.Message
}

type Service struct {
	storage storage.Storage
	rand    *rand.Rand // for selecting reviewers
}

func NewService(storage storage.Storage) *Service {
	source := rand.NewSource(time.Now().UnixNano())
	return &Service{
		storage: storage,
		rand:    rand.New(source),
	}
}

// TEAMS

func (s *Service) CreateTeam(req *models.TeamResponse) error {
	exists, err := s.storage.TeamExists(req.TeamName)
	if err != nil {
		return err
	}
	if exists {
		return &ServiceError{
			Code:    "TEAM_EXISTS",
			Message: "team already exists",
		}
	}
	
	if err := s.storage.CreateTeam(req.TeamName); err != nil {
		return err
	}
	
	for _, member := range req.Members {
		user := &models.User{
			UserID:   member.UserID,
			Username: member.Username,
			TeamName: req.TeamName,
			IsActive: member.IsActive,
		}
		if err := s.storage.CreateOrUpdateUser(user); err != nil {
			return err
		}
	}
	
	return nil
}

func (s *Service) GetTeam(teamName string) (*models.TeamResponse, error) {
	team, err := s.storage.GetTeam(teamName)
	if err != nil {
		return nil, &ServiceError{
			Code:    "NOT_FOUND",
			Message: "team not found",
		}
	}
	return team, nil
}

// USERS

func (s *Service) SetUserActive(userID string, isActive bool) (*models.User, error) {
	user, err := s.storage.GetUser(userID)
	if err != nil {
		return nil, &ServiceError{
			Code:    "NOT_FOUND",
			Message: "user not found",
		}
	}
	
	if err := s.storage.SetUserActive(userID, isActive); err != nil {
		return nil, err
	}
	
	user.IsActive = isActive
	return user, nil
}

func (s *Service) GetPRsByReviewer(userID string) ([]models.PullRequestShort, error) {
	_, err := s.storage.GetUser(userID)
	if err != nil {
		return nil, &ServiceError{
			Code:    "NOT_FOUND",
			Message: "user not found",
		}
	}
	
	prs, err := s.storage.GetPRsByReviewer(userID)
	if err != nil {
		return nil, err
	}
	
	return prs, nil
}

// PULL REQUESTS

// CreatePullRequest creates PR and automatically assigns up to 2 reviewers
func (s *Service) CreatePullRequest(prID, prName, authorID string) (*models.PullRequest, error) {
	exists, err := s.storage.PRExists(prID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, &ServiceError{
			Code:    "PR_EXISTS",
			Message: "pull request already exists",
		}
	}
	
	author, err := s.storage.GetUser(authorID)
	if err != nil {
		return nil, &ServiceError{
			Code:    "NOT_FOUND",
			Message: "author not found",
		}
	}
	
	pr := &models.PullRequest{
		PullRequestID:   prID,
		PullRequestName: prName,
		AuthorID:        authorID,
		Status:          "OPEN",
		CreatedAt:       time.Now(),
	}
	
	if err := s.storage.CreatePullRequest(pr); err != nil {
		return nil, err
	}
	
	reviewers, err := s.assignReviewers(author.TeamName, authorID, 2)
	if err != nil {
		return nil, err
	}
	
	for _, reviewerID := range reviewers {
		if err := s.storage.AddReviewer(prID, reviewerID); err != nil {
			return nil, err
		}
	}
	
	pr.AssignedReviewers = reviewers
	return pr, nil
}

// assignReviewers selects random active team members
func (s *Service) assignReviewers(teamName, excludeUserID string, maxCount int) ([]string, error) {
	candidates, err := s.storage.GetActiveTeamMembers(teamName, excludeUserID)
	if err != nil {
		return nil, err
	}
	
	count := maxCount
	if len(candidates) < count {
		count = len(candidates)
	}
	
	selected := make([]string, 0, count)
	
	s.rand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})
	
	for i := 0; i < count; i++ {
		selected = append(selected, candidates[i].UserID)
	}
	
	return selected, nil
}

func (s *Service) MergePullRequest(prID string) (*models.PullRequest, error) {
	if err := s.storage.MergePullRequest(prID); err != nil {
		return nil, err
	}
	
	pr, err := s.storage.GetPullRequest(prID)
	if err != nil {
		return nil, err
	}
	
	return pr, nil
}

func (s *Service) ReassignReviewer(prID, oldReviewerID string) (*models.PullRequest, string, error) {
	pr, err := s.storage.GetPullRequest(prID)
	if err != nil {
		return nil, "", &ServiceError{
			Code:    "NOT_FOUND",
			Message: "pull request not found",
		}
	}
	
	if pr.Status == "MERGED" {
		return nil, "", &ServiceError{
			Code:    "PR_MERGED",
			Message: "cannot reassign on merged PR",
		}
	}
	
	isAssigned, err := s.storage.IsReviewerAssigned(prID, oldReviewerID)
	if err != nil {
		return nil, "", err
	}
	if !isAssigned {
		return nil, "", &ServiceError{
			Code:    "NOT_ASSIGNED",
			Message: "user is not assigned as reviewer to this PR",
		}
	}
	
	oldReviewer, err := s.storage.GetUser(oldReviewerID)
	if err != nil {
		return nil, "", &ServiceError{
			Code:    "NOT_FOUND",
			Message: "reviewer not found",
		}
	}
	
	candidates, err := s.storage.GetActiveTeamMembers(oldReviewer.TeamName, oldReviewerID)
	if err != nil {
		return nil, "", err
	}
	
	// Exclude current reviewers and author from candidates
	var availableCandidates []models.User
	for _, candidate := range candidates {
		if candidate.UserID == pr.AuthorID {
			continue
		}
		isAlreadyAssigned, err := s.storage.IsReviewerAssigned(prID, candidate.UserID)
		if err != nil {
			return nil, "", err
		}
		if !isAlreadyAssigned {
			availableCandidates = append(availableCandidates, candidate)
		}
	}
	
	if len(availableCandidates) == 0 {
		return nil, "", &ServiceError{
			Code:    "NO_CANDIDATE",
			Message: "no active replacement candidate available in team",
		}
	}
	
	// Select random candidate
	newReviewerID := availableCandidates[s.rand.Intn(len(availableCandidates))].UserID
	
	if err := s.storage.RemoveReviewer(prID, oldReviewerID); err != nil {
		return nil, "", err
	}
	if err := s.storage.AddReviewer(prID, newReviewerID); err != nil {
		return nil, "", err
	}
	
	pr, err = s.storage.GetPullRequest(prID)
	if err != nil {
		return nil, "", err
	}
	
	return pr, newReviewerID, nil
}
