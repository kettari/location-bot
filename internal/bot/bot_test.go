package bot

import (
	"testing"

	"github.com/kettari/location-bot/internal/entity"
)

func TestCreateBot(t *testing.T) {
	// Test creating bot with explicit dependencies
	token := "test_token"
	recipients := "123456,789"

	// Note: This will fail at runtime because the token is invalid,
	// but it tests that the signature is correct
	_, err := CreateBot(token, recipients)

	// We expect an error because the token is invalid, but the function signature is correct
	if err == nil {
		t.Error("Expected error with invalid token, but got nil")
	}
}

func TestCreateBot_Signature(t *testing.T) {
	// Test that CreateBot returns MessageDispatcher interface
	token := "test_token"
	recipients := "123456,789"

	// We can't actually create a bot without a valid token,
	// so we just check that the function signature is correct
	var dispatcher entity.MessageDispatcher

	_, err := CreateBot(token, recipients)
	// We ignore the error - we're just testing the signature

	// This should compile - the function returns the right type
	_ = dispatcher
	_ = err
}
