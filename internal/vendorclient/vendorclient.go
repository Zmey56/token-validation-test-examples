package vendorclient

import (
	"context"
	"database/sql"
)

type VendorClient interface {
	ValidateToken(token string) (bool, error)
}

type TokenValidator struct {
	db     *sql.DB
	client VendorClient
}

func NewTokenValidator(db *sql.DB, client VendorClient) *TokenValidator {
	return &TokenValidator{
		db:     db,
		client: client,
	}
}

func (tv *TokenValidator) ValidateUserToken(ctx context.Context, userID int, token string) (bool, error) {
	// Step 1: Checking for a token validation record in the database
	var validated bool
	err := tv.db.QueryRowContext(ctx, "SELECT validated FROM tokens WHERE user_id = $1 AND token = $2", userID, token).Scan(&validated)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}

	// If the token has already been validated, we return the result
	if validated && err == nil {
		return true, nil
	}

	// Step 2: The token is not validated, we contact the API vendors
	validated, err = tv.client.ValidateToken(token)
	if err != nil {
		return false, err
	}

	// Step 3: Save the result of the token validation in the database
	_, err = tv.db.ExecContext(ctx, "INSERT INTO tokens (user_id, token, validated) VALUES ($1, $2, $3) ON CONFLICT (user_id, token) DO UPDATE SET validated = $3", userID, token, validated)
	if err != nil {
		return false, err
	}

	return validated, nil

}
