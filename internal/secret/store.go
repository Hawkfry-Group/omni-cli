package secret

import (
	"errors"
	"fmt"
)

type Store interface {
	Name() string
	Available() bool
	Save(profileName, token string) (string, error)
	Get(ref string) (string, error)
	Delete(ref string) error
}

type ConfigStore struct{}

func NewConfigStore() Store { return ConfigStore{} }

func (ConfigStore) Name() string { return "config" }

func (ConfigStore) Available() bool { return true }

func (ConfigStore) Save(profileName, token string) (string, error) {
	if token == "" {
		return "", errors.New("token is empty")
	}
	return "", nil
}

func (ConfigStore) Get(ref string) (string, error) {
	return "", errors.New("config store does not support token references")
}

func (ConfigStore) Delete(ref string) error { return nil }

func Pick(preference string, keychain Store) (Store, error) {
	switch preference {
	case "", "auto":
		if keychain != nil && keychain.Available() {
			return keychain, nil
		}
		return NewConfigStore(), nil
	case "config":
		return NewConfigStore(), nil
	case "keychain":
		if keychain == nil || !keychain.Available() {
			return nil, errors.New("keychain store is unavailable on this system")
		}
		return keychain, nil
	default:
		return nil, fmt.Errorf("invalid token store %q; use auto, keychain, or config", preference)
	}
}
