package service

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/sony/gobreaker"
)

// EmailService мокает отправку email при приглашении в команду.
// Обёрнут в circuit breaker, т.к. в реальности это вызов внешнего сервиса.
type EmailService struct {
	breaker *gobreaker.CircuitBreaker
}

func NewEmailService() *EmailService {
	settings := gobreaker.Settings{
		Name:        "email-service",
		MaxRequests: 3,
		Interval:    30 * time.Second,
		Timeout:     10 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
	}
	return &EmailService{breaker: gobreaker.NewCircuitBreaker(settings)}
}

func (s *EmailService) SendInvite(ctx context.Context, toEmail, teamName string) error {
	_, err := s.breaker.Execute(func() (interface{}, error) {
		return nil, s.send(toEmail, teamName)
	})
	if errors.Is(err, gobreaker.ErrOpenState) {
		log.Printf("email service circuit open, skipping invite to %s", toEmail)
		return nil
	}
	return err
}

func (s *EmailService) send(toEmail, teamName string) error {
	log.Printf("[email-mock] invite sent to %s for team %s", toEmail, teamName)
	return nil
}
