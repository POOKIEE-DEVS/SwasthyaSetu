package api

import (
	"context"

	"github.com/POOKIEE-DEVS/SwasthyaSetu/backend/internal/store"
)

// The JSON shapes the frontend reads (frontend/lib/api.ts). Times are Unix
// seconds, so browsers never misread them as local time.

type verificationSummary struct {
	Role            string  `json:"role"`
	Status          string  `json:"status"`
	RejectionReason *string `json:"rejection_reason"`
	FullName        string  `json:"full_name"`
}

type userPublic struct {
	ID         int64   `json:"id"`
	Email      string  `json:"email"`
	Name       string  `json:"name"`
	PictureURL *string `json:"picture_url"`
	Role       *string `json:"role"`
	IsAdmin    bool    `json:"is_admin"`
	// Professionals only: where their verification stands.
	Verification *verificationSummary `json:"verification"`
}

type meResponse struct {
	// nil when signed out. Always 200, so pages can check quietly.
	User          *userPublic `json:"user"`
	GoogleEnabled bool        `json:"google_enabled"`
	DevLogin      bool        `json:"dev_login"`
}

type documentInfo struct {
	ID          int64  `json:"id"`
	Kind        string `json:"kind"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

type applicationPublic struct {
	ID                  int64          `json:"id"`
	Role                string         `json:"role"`
	FullName            string         `json:"full_name"`
	Phone               string         `json:"phone"`
	CitizenshipNumber   string         `json:"citizenship_number"`
	CitizenshipDistrict string         `json:"citizenship_district"`
	CouncilNumber       *string        `json:"council_number"`
	Institution         *string        `json:"institution"`
	RecommenderName     *string        `json:"recommender_name"`
	RecommenderNMC      *string        `json:"recommender_nmc"`
	Status              string         `json:"status"`
	RejectionReason     *string        `json:"rejection_reason"`
	SubmittedAt         float64        `json:"submitted_at"`
	ReviewedAt          *float64       `json:"reviewed_at"`
	Documents           []documentInfo `json:"documents"`
}

type auditPublic struct {
	Action     string  `json:"action"`
	ActorEmail string  `json:"actor_email"`
	Reason     *string `json:"reason"`
	At         float64 `json:"at"`
}

// adminApplication is the admin's review card: the application, the account
// and the history.
type adminApplication struct {
	applicationPublic
	UserEmail      string        `json:"user_email"`
	UserName       string        `json:"user_name"`
	UserPictureURL *string       `json:"user_picture_url"`
	History        []auditPublic `json:"history"`
}

func (s *Server) userPublic(ctx context.Context, u *store.User) (*userPublic, error) {
	application, err := s.store.ApplicationForUser(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	view := &userPublic{
		ID: u.ID, Email: u.Email, Name: u.Name, PictureURL: u.PictureURL, Role: u.Role,
		IsAdmin: s.cfg.IsAdmin(u.Email),
	}
	if application != nil {
		view.Verification = &verificationSummary{
			Role: application.Role, Status: application.Status,
			RejectionReason: application.RejectionReason, FullName: application.FullName,
		}
	}
	return view, nil
}

func (s *Server) applicationPublic(ctx context.Context, a *store.Application) (*applicationPublic, error) {
	docs, err := s.store.Documents(ctx, a.ID)
	if err != nil {
		return nil, err
	}
	view := &applicationPublic{
		ID: a.ID, Role: a.Role, FullName: a.FullName, Phone: a.Phone,
		CitizenshipNumber: a.CitizenshipNumber, CitizenshipDistrict: a.CitizenshipDistrict,
		CouncilNumber: a.CouncilNumber, Institution: a.Institution,
		RecommenderName: a.RecommenderName, RecommenderNMC: a.RecommenderNMC,
		Status: a.Status, RejectionReason: a.RejectionReason,
		SubmittedAt: store.Unix(a.SubmittedAt), Documents: []documentInfo{},
	}
	if a.ReviewedAt != nil {
		reviewed := store.Unix(*a.ReviewedAt)
		view.ReviewedAt = &reviewed
	}
	for _, d := range docs {
		view.Documents = append(view.Documents, documentInfo{ID: d.ID, Kind: d.Kind, ContentType: d.ContentType, Size: d.Size})
	}
	return view, nil
}

func (s *Server) adminApplication(ctx context.Context, a *store.Application) (*adminApplication, error) {
	public, err := s.applicationPublic(ctx, a)
	if err != nil {
		return nil, err
	}
	user, err := s.store.UserByID(ctx, a.UserID)
	if err != nil {
		return nil, err
	}
	history, err := s.store.History(ctx, a.ID)
	if err != nil {
		return nil, err
	}
	view := &adminApplication{applicationPublic: *public, History: []auditPublic{}}
	if user != nil {
		view.UserEmail, view.UserName, view.UserPictureURL = user.Email, user.Name, user.PictureURL
	}
	for _, e := range history {
		view.History = append(view.History, auditPublic{
			Action: e.Action, ActorEmail: e.ActorEmail, Reason: e.Reason, At: store.Unix(e.At),
		})
	}
	return view, nil
}
