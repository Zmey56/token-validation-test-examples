package tests

import (
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/zmey56/token-validation-test-examples/tests/mocks"
)

func TestValidateToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockVendorClient := mocks.NewMockVendorClient(ctrl)

	// Scenario: successful token validation
	mockVendorClient.EXPECT().ValidateToken("valid_token").Return(true, nil)
	success, err := mockVendorClient.ValidateToken("valid_token")
	if err != nil || !success {
		t.Fatalf("Expected successful validation, but there was an error or the token was not validated")
	}

	// Scenario: unsuccessful token validation
	mockVendorClient.EXPECT().ValidateToken("invalid_token").Return(false, nil)
	success, err = mockVendorClient.ValidateToken("invalid_token")
	if err != nil || success {
		t.Fatalf("Expected unsuccessful validation, but there was an error or the token was validated")
	}
}
