//go:build !darwin

package secret

type unavailableKeychainStore struct{}

func NewKeychainStore() Store { return unavailableKeychainStore{} }

func (unavailableKeychainStore) Name() string { return "keychain" }

func (unavailableKeychainStore) Available() bool { return false }

func (unavailableKeychainStore) Save(profileName, token string) (string, error) {
	return "", errUnavailable()
}

func (unavailableKeychainStore) Get(ref string) (string, error) { return "", errUnavailable() }

func (unavailableKeychainStore) Delete(ref string) error { return nil }

func errUnavailable() error { return &unsupportedError{} }

type unsupportedError struct{}

func (unsupportedError) Error() string { return "keychain store is unavailable on this system" }
