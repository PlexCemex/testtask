package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEmailService_SendInvite(t *testing.T) {
	svc := NewEmailService()

	err := svc.SendInvite(context.Background(), "user@example.com", "Dev Team")
	require.NoError(t, err)
}
