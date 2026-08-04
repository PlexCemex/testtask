package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTManager_GenerateAndVerify(t *testing.T) {
	m := NewJWTManager("test-secret", 60)

	token, err := m.Generate(42, "user@example.com")
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := m.Verify(token)
	require.NoError(t, err)
	assert.Equal(t, int64(42), claims.UserID)
	assert.Equal(t, "user@example.com", claims.Email)
}

func TestJWTManager_VerifyInvalidToken(t *testing.T) {
	m := NewJWTManager("test-secret", 60)

	_, err := m.Verify("not-a-valid-token")
	assert.Error(t, err)
}

func TestJWTManager_VerifyWrongSecret(t *testing.T) {
	m1 := NewJWTManager("secret-one", 60)
	m2 := NewJWTManager("secret-two", 60)

	token, err := m1.Generate(1, "a@b.com")
	require.NoError(t, err)

	_, err = m2.Verify(token)
	assert.Error(t, err)
}

func TestJWTManager_ExpiredToken(t *testing.T) {
	m := NewJWTManager("test-secret", 0)

	token, err := m.Generate(1, "a@b.com")
	require.NoError(t, err)

	time.Sleep(1100 * time.Millisecond)

	_, err = m.Verify(token)
	assert.Error(t, err)
}
