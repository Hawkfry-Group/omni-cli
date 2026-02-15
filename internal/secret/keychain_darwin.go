//go:build darwin

package secret

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

const keychainService = "omni-cli"

type KeychainStore struct{}

func NewKeychainStore() Store { return KeychainStore{} }

func (KeychainStore) Name() string { return "keychain" }

func (KeychainStore) Available() bool {
	_, err := exec.LookPath("security")
	return err == nil
}

func (KeychainStore) Save(profileName, token string) (string, error) {
	if strings.TrimSpace(profileName) == "" {
		return "", errors.New("profile name is empty")
	}
	if strings.TrimSpace(token) == "" {
		return "", errors.New("token is empty")
	}

	cmd := exec.Command("security", "add-generic-password", "-U", "-s", keychainService, "-a", profileName, "-w", token)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("save token to keychain: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return profileName, nil
}

func (KeychainStore) Get(ref string) (string, error) {
	if strings.TrimSpace(ref) == "" {
		return "", errors.New("token reference is empty")
	}
	cmd := exec.Command("security", "find-generic-password", "-s", keychainService, "-a", ref, "-w")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("read token from keychain: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(bytes.TrimSpace(out))), nil
}

func (KeychainStore) Delete(ref string) error {
	if strings.TrimSpace(ref) == "" {
		return nil
	}
	cmd := exec.Command("security", "delete-generic-password", "-s", keychainService, "-a", ref)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return nil
	}
	return nil
}
