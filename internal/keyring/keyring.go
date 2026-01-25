package keyring

import (
	"errors"
	"fmt"
	"strings"

	gokeyring "github.com/zalando/go-keyring"
)

const (
	KeyAccessToken  = "access_token"
	KeyRefreshToken = "refresh_token"
	KeyAPIKey       = "api_key"
)

var (
	ErrNotFound    = errors.New("keyring: secret not found")
	ErrUnavailable = errors.New("keyring: service unavailable")
)

// Store defines the interface for secure secret storage.
type Store interface {
	Set(key, value string) error
	Get(key string) (string, error)
	Delete(key string) error
	DeleteAll() error
	Available() bool
}

type OSKeyring struct {
	service string
}

var _ Store = (*OSKeyring)(nil)

func NewOSKeyring() *OSKeyring {
	return &OSKeyring{service: ServiceName}
}

func (k *OSKeyring) Set(key, value string) error {
	if err := gokeyring.Set(k.service, key, value); err != nil {
		if isUnavailableError(err) {
			return fmt.Errorf("%w: %v", ErrUnavailable, err)
		}
		return fmt.Errorf("keyring set failed: %w", err)
	}
	return nil
}

func (k *OSKeyring) Get(key string) (string, error) {
	value, err := gokeyring.Get(k.service, key)
	if err != nil {
		if errors.Is(err, gokeyring.ErrNotFound) {
			return "", ErrNotFound
		}
		if isUnavailableError(err) {
			return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
		}
		return "", fmt.Errorf("keyring get failed: %w", err)
	}
	return value, nil
}

func (k *OSKeyring) Delete(key string) error {
	if err := gokeyring.Delete(k.service, key); err != nil {
		if errors.Is(err, gokeyring.ErrNotFound) {
			return nil
		}
		if isUnavailableError(err) {
			return fmt.Errorf("%w: %v", ErrUnavailable, err)
		}
		return fmt.Errorf("keyring delete failed: %w", err)
	}
	return nil
}

func (k *OSKeyring) DeleteAll() error {
	keys := []string{KeyAccessToken, KeyRefreshToken, KeyAPIKey}
	for _, key := range keys {
		if err := k.Delete(key); err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
	}
	return nil
}

func (k *OSKeyring) Available() bool {
	testKey := "_thoop_availability_check"
	if err := gokeyring.Set(k.service, testKey, "test"); err != nil {
		return false
	}
	_ = gokeyring.Delete(k.service, testKey)
	return true
}

func isUnavailableError(err error) bool {
	errStr := err.Error()
	return strings.Contains(errStr, "secret service") ||
		strings.Contains(errStr, "dbus") ||
		strings.Contains(errStr, "no keyring")
}
