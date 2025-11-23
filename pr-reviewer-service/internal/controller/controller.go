package controller

import (
	"encoding/json"
	"net/http"
	"pr-reviewer-service/internal/models"
	"pr-reviewer-service/internal/service"
	"log"
)

type Controller struct {
	service *service.Service
}

func NewController(service *service.Service) *Controller {
	return &Controller{
		service: service,
	}
}

func (c *Controller) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
	}
}

func (c *Controller) respondError(w http.ResponseWriter, status int, code, message string) {
	c.respondJSON(w, status, models.ErrorResponse{
		Error: models.ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

func (c *Controller) parseJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// TEAMS

// CreateTeam - POST /team/add
func (c *Controller) CreateTeam(w http.ResponseWriter, r *http.Request) {
	var req models.TeamResponse
	if err := c.parseJSON(r, &req); err != nil {
		c.respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON")
		return
	}
	
	if err := c.service.CreateTeam(&req); err != nil {
		if serviceErr, ok := err.(*service.ServiceError); ok {
			switch serviceErr.Code {
			case "TEAM_EXISTS":
				c.respondError(w, http.StatusBadRequest, serviceErr.Code, serviceErr.Message)
			default:
				c.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", serviceErr.Message)
			}
			return
		}
		c.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	
	c.respondJSON(w, http.StatusCreated, map[string]interface{}{
		"team": req,
	})
}

// GetTeam - GET /team/get
func (c *Controller) GetTeam(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		c.respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "team_name is required")
		return
	}
	
	team, err := c.service.GetTeam(teamName)
	if err != nil {
		if serviceErr, ok := err.(*service.ServiceError); ok {
			if serviceErr.Code == "NOT_FOUND" {
				c.respondError(w, http.StatusNotFound, serviceErr.Code, serviceErr.Message)
				return
			}
		}
		c.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	
	c.respondJSON(w, http.StatusOK, team)
}

// USERS

// SetUserActive - POST /users/setIsActive
func (c *Controller) SetUserActive(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string `json:"user_id"`
		IsActive bool   `json:"is_active"`
	}
	
	if err := c.parseJSON(r, &req); err != nil {
		c.respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON")
		return
	}
	
	user, err := c.service.SetUserActive(req.UserID, req.IsActive)
	if err != nil {
		if serviceErr, ok := err.(*service.ServiceError); ok {
			if serviceErr.Code == "NOT_FOUND" {
				c.respondError(w, http.StatusNotFound, serviceErr.Code, serviceErr.Message)
				return
			}
		}
		c.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	
	c.respondJSON(w, http.StatusOK, map[string]interface{}{
		"user": user,
	})
}

// GetUserReviews - GET /users/getReview
func (c *Controller) GetUserReviews(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		c.respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "user_id is required")
		return
	}
	
	prs, err := c.service.GetPRsByReviewer(userID)
	if err != nil {
		if serviceErr, ok := err.(*service.ServiceError); ok {
			if serviceErr.Code == "NOT_FOUND" {
				c.respondError(w, http.StatusNotFound, serviceErr.Code, serviceErr.Message)
				return
			}
		}
		c.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	
	c.respondJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":       userID,
		"pull_requests": prs,
	})
}

// PULL REQUESTS

// CreatePullRequest - POST /pullRequest/create
func (c *Controller) CreatePullRequest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PullRequestID   string `json:"pull_request_id"`
		PullRequestName string `json:"pull_request_name"`
		AuthorID        string `json:"author_id"`
	}
	
	if err := c.parseJSON(r, &req); err != nil {
		c.respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON")
		return
	}
	
	pr, err := c.service.CreatePullRequest(req.PullRequestID, req.PullRequestName, req.AuthorID)
	if err != nil {
		if serviceErr, ok := err.(*service.ServiceError); ok {
			switch serviceErr.Code {
			case "PR_EXISTS":
				c.respondError(w, http.StatusConflict, serviceErr.Code, serviceErr.Message)
			case "NOT_FOUND":
				c.respondError(w, http.StatusNotFound, serviceErr.Code, serviceErr.Message)
			default:
				c.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", serviceErr.Message)
			}
			return
		}
		c.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	
	c.respondJSON(w, http.StatusCreated, map[string]interface{}{
		"pr": pr,
	})
}

// MergePullRequest - POST /pullRequest/merge
func (c *Controller) MergePullRequest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PullRequestID string `json:"pull_request_id"`
	}
	
	if err := c.parseJSON(r, &req); err != nil {
		c.respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON")
		return
	}
	
	pr, err := c.service.MergePullRequest(req.PullRequestID)
	if err != nil {
		if serviceErr, ok := err.(*service.ServiceError); ok {
			if serviceErr.Code == "NOT_FOUND" {
				c.respondError(w, http.StatusNotFound, serviceErr.Code, serviceErr.Message)
				return
			}
		}
		c.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	
	c.respondJSON(w, http.StatusOK, map[string]interface{}{
		"pr": pr,
	})
}

// ReassignReviewer - POST /pullRequest/reassign
func (c *Controller) ReassignReviewer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PullRequestID string `json:"pull_request_id"`
		OldUserID     string `json:"old_user_id"`
	}
	
	if err := c.parseJSON(r, &req); err != nil {
		c.respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON")
		return
	}
	
	pr, newReviewerID, err := c.service.ReassignReviewer(req.PullRequestID, req.OldUserID)
	if err != nil {
		if serviceErr, ok := err.(*service.ServiceError); ok {
			switch serviceErr.Code {
			case "NOT_FOUND":
				c.respondError(w, http.StatusNotFound, serviceErr.Code, serviceErr.Message)
			case "PR_MERGED", "NOT_ASSIGNED", "NO_CANDIDATE":
				c.respondError(w, http.StatusConflict, serviceErr.Code, serviceErr.Message)
			default:
				c.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", serviceErr.Message)
			}
			return
		}
		c.respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	
	c.respondJSON(w, http.StatusOK, map[string]interface{}{
		"pr":          pr,
		"replaced_by": newReviewerID,
	})
}