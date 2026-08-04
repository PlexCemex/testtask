package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("mypassword123")
	require.NoError(t, err)
	assert.NotEqual(t, "mypassword123", hash)

	assert.True(t, CheckPassword(hash, "mypassword123"))
	assert.False(t, CheckPassword(hash, "wrongpassword"))
}
