package tests

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/zmey56/token-validation-test-examples/internal/vendorclient"
	"github.com/zmey56/token-validation-test-examples/tests/mocks"
)

// createTestDBContainer starts a PostgreSQL container and returns a connection URL
func createTestDBContainer(ctx context.Context) (string, func(), error) {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:latest",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "testuser",
			"POSTGRES_PASSWORD": "testpass",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(60 * time.Second),
	}

	postgresContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return "", nil, err
	}

	port, err := postgresContainer.MappedPort(ctx, "5432")
	if err != nil {
		return "", nil, err
	}

	host, err := postgresContainer.Host(ctx)
	if err != nil {
		return "", nil, err
	}

	dbURL := fmt.Sprintf("postgres://testuser:testpass@%s:%s/testdb?sslmode=disable", host, port.Port())
	cleanup := func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			log.Fatalf("Error terminating the container: %v", err)
		}
	}

	return dbURL, cleanup, nil
}

// TestDatabaseConnection checks database connection and table initialization
func TestDatabaseConnection(t *testing.T) {
	ctx := context.Background()
	dbURL, cleanup, err := createTestDBContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to create PostgreSQL container: %v", err)
	}
	defer cleanup()

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to the database: %v", err)
	}
	defer db.Close()

	// Check database connection
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("Error while connecting to the database: %v", err)
	}

	// Execute database initialization: create tokens table
	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS tokens (id SERIAL PRIMARY KEY, token VARCHAR(255) NOT NULL, validated BOOLEAN NOT NULL);`)
	if err != nil {
		t.Fatalf("Error initializing the database: %v", err)
	}

	t.Log("Database connection and initialization were successful.")
}

func setupTestContainer(t *testing.T) (*sql.DB, func()) {
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "postgres:latest",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "testuser",
			"POSTGRES_PASSWORD": "testpass",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(60 * time.Second),
	}

	postgresContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("Не удалось создать контейнер PostgreSQL: %v", err)
	}

	port, _ := postgresContainer.MappedPort(ctx, "5432")
	host, _ := postgresContainer.Host(ctx)
	dbURL := "postgres://testuser:testpass@" + host + ":" + port.Port() + "/testdb?sslmode=disable"

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		t.Fatalf("Не удалось подключиться к базе данных: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS tokens (user_id INT, token VARCHAR(255) NOT NULL, validated BOOLEAN NOT NULL, PRIMARY KEY (user_id, token));`)
	if err != nil {
		t.Fatalf("Ошибка при выполнении миграций: %v", err)
	}

	return db, func() {
		db.Close()
		postgresContainer.Terminate(ctx)
	}
}

func TestTokenValidationIntegration(t *testing.T) {
	db, cleanup := setupTestContainer(t)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockVendorClient := mocks.NewMockVendorClient(ctrl)
	validator := vendorclient.NewTokenValidator(db, mockVendorClient)

	ctx := context.Background()
	userID := 1
	token := "test_token"

	// Scenario 1: Availability of a validated token
	_, err := db.ExecContext(ctx, "INSERT INTO tokens (user_id, token, validated) VALUES ($1, $2, $3)", userID, token, true)
	if err != nil {
		t.Fatalf("Error when adding a token to the database: %v", err)
	}

	validated, err := validator.ValidateUserToken(ctx, userID, token)
	if err != nil || !validated {
		t.Errorf("Successful validation was expected if there was a record")
	}

	// Scenario 2: A new token requiring vendor validation
	newToken := "new_token"
	mockVendorClient.EXPECT().ValidateToken(newToken).Return(true, nil)
	validated, err = validator.ValidateUserToken(ctx, userID, newToken)
	if err != nil || !validated {
		t.Errorf("Successful validation of the new token was expected")
	}

	// Scenario 3: Re-checking the saved token
	validated, err = validator.ValidateUserToken(ctx, userID, newToken)
	if err != nil || !validated {
		t.Errorf("Successful re-validation of the saved token was expected")
	}
}
