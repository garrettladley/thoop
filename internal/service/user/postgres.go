package user

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"

	pgc "github.com/garrettladley/thoop/internal/sqlc/postgres"
	"github.com/jackc/pgx/v5"
)

type PostgresService struct {
	db pgc.Querier
}

var _ Service = (*PostgresService)(nil)

func NewPostgresService(db pgc.Querier) *PostgresService {
	return &PostgresService{db: db}
}

func (s *PostgresService) ValidateAPIKey(ctx context.Context, apiKey string) (*ValidatedUser, error) {
	keyHash := hashSecret(apiKey)

	apiKeyRecord, err := s.db.GetAPIKeyByHash(ctx, keyHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAPIKeyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("getting API key by hash: %w", err)
	}

	if apiKeyRecord.Revoked {
		return nil, ErrAPIKeyRevoked
	}

	user, err := s.db.GetUser(ctx, apiKeyRecord.WhoopUserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	if user.Banned {
		return nil, ErrUserBanned
	}

	return &ValidatedUser{
		WhoopUserID: apiKeyRecord.WhoopUserID,
		APIKeyID:    apiKeyRecord.ID,
	}, nil
}

func (s *PostgresService) GetOrCreateUser(ctx context.Context, whoopUserID int64) (string, bool, error) {
	user, err := s.db.GetUser(ctx, whoopUserID)
	if err == nil {
		return "", user.Banned, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return "", false, fmt.Errorf("getting user: %w", err)
	}

	user, err = s.db.CreateUser(ctx, whoopUserID)
	if err != nil {
		return "", false, fmt.Errorf("creating user: %w", err)
	}

	apiKey, err := generateAPIKey()
	if err != nil {
		return "", false, err
	}

	keyHash := hashSecret(apiKey)
	keyName := "default"
	_, err = s.db.CreateAPIKey(ctx, pgc.CreateAPIKeyParams{
		WhoopUserID: whoopUserID,
		KeyHash:     keyHash,
		Name:        &keyName,
	})
	if err != nil {
		return "", false, fmt.Errorf("creating API key: %w", err)
	}

	return apiKey, user.Banned, nil
}

func (s *PostgresService) UpdateAPIKeyLastUsed(ctx context.Context, apiKeyID int64) error {
	if err := s.db.UpdateAPIKeyLastUsed(ctx, apiKeyID); err != nil {
		return fmt.Errorf("updating API key last used: %w", err)
	}
	return nil
}

func (s *PostgresService) IsBanned(ctx context.Context, whoopUserID int64) (bool, error) {
	user, err := s.db.GetUser(ctx, whoopUserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, ErrUserNotFound
	}
	if err != nil {
		return false, fmt.Errorf("getting user: %w", err)
	}
	return user.Banned, nil
}

func hashSecret(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

const (
	apiKeyPrefix = "thp_"
	apiKeyLength = 32
)

func generateAPIKey() (string, error) {
	b := make([]byte, apiKeyLength)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating random bytes: %w", err)
	}
	return apiKeyPrefix + base64.RawURLEncoding.EncodeToString(b), nil
}
