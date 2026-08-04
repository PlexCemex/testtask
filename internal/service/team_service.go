package service

import (
	"context"
	"database/sql"
	"errors"

	"taskservice/internal/models"
	"taskservice/internal/repository"
)

var ErrForbidden = errors.New("forbidden")

type TeamService struct {
	teams *repository.TeamRepository
	users *repository.UserRepository
	email *EmailService
}

func NewTeamService(teams *repository.TeamRepository, users *repository.UserRepository, email *EmailService) *TeamService {
	return &TeamService{teams: teams, users: users, email: email}
}

func (s *TeamService) Create(ctx context.Context, name string, ownerID int64) (*models.Team, error) {
	var teamID int64
	err := s.teams.WithTx(ctx, func(tx *sql.Tx) error {
		id, err := s.teams.Create(ctx, tx, name, ownerID)
		if err != nil {
			return err
		}
		teamID = id
		return s.teams.AddMember(ctx, tx, teamID, ownerID, models.RoleOwner)
	})
	if err != nil {
		return nil, err
	}

	return s.teams.GetByID(ctx, teamID)
}

func (s *TeamService) ListForUser(ctx context.Context, userID int64) ([]models.Team, error) {
	return s.teams.ListByUser(ctx, userID)
}

func (s *TeamService) Invite(ctx context.Context, teamID, inviterID int64, inviteeEmail string) error {
	role, err := s.teams.GetMemberRole(ctx, teamID, inviterID)
	if err != nil {
		return err
	}
	if role != models.RoleOwner && role != models.RoleAdmin {
		return ErrForbidden
	}

	invitee, err := s.users.GetByEmail(ctx, inviteeEmail)
	if err != nil {
		return err
	}

	err = s.teams.WithTx(ctx, func(tx *sql.Tx) error {
		return s.teams.AddMember(ctx, tx, teamID, invitee.ID, models.RoleMember)
	})
	if err != nil {
		return err
	}

	team, err := s.teams.GetByID(ctx, teamID)
	if err != nil {
		return err
	}

	return s.email.SendInvite(ctx, invitee.Email, team.Name)
}

func (s *TeamService) TeamStats(ctx context.Context, userID int64) ([]models.TeamStats, error) {
	return s.teams.TeamStats(ctx, userID)
}

func (s *TeamService) TopCreators(ctx context.Context, teamID int64) ([]models.TopCreator, error) {
	return s.teams.TopCreators(ctx, teamID)
}

func (s *TeamService) OrphanAssignees(ctx context.Context, teamID int64) ([]models.Task, error) {
	return s.teams.OrphanAssigneeTasks(ctx, teamID)
}
