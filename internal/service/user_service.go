package service

import (
	"context"
	"errors"

	"taskservice/internal/auth"
	"taskservice/internal/models"
	"taskservice/internal/repository"
)

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrEmailTaken = errors.New("email already registered")

type UserService struct {
	users *repository.UserRepository
	jwt   *auth.JWTManager
}

func NewUserService(users *repository.UserRepository, jwt *auth.JWTManager) *UserService {
	return &UserService{users: users, jwt: jwt}
}

func (s *UserService) Register(ctx context.Context, email, password, name string) (*models.User, error) {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}

	u := &models.User{Email: email, PasswordHash: hash, Name: name}
	id, err := s.users.Create(ctx, u)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return nil, ErrEmailTaken
		}
		return nil, err
	}
	u.ID = id
	return u, nil
}

func (s *UserService) Login(ctx context.Context, email, password string) (string, error) {
	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", ErrInvalidCredentials
		}
		return "", err
	}

	if !auth.CheckPassword(u.PasswordHash, password) {
		return "", ErrInvalidCredentials
	}

	return s.jwt.Generate(u.ID, u.Email)
}
