package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Profile struct {
	BaseURL     string `json:"base_url"`
	DefaultAuth string `json:"default_auth,omitempty"`
	PATToken    string `json:"pat_token,omitempty"`
	PATStore    string `json:"pat_token_store,omitempty"`
	PATRef      string `json:"pat_token_ref,omitempty"`
	OrgKey      string `json:"org_key,omitempty"`
	OrgKeyStore string `json:"org_key_store,omitempty"`
	OrgKeyRef   string `json:"org_key_ref,omitempty"`

	// Legacy single-token fields kept for backward-compatible reads.
	Token      string `json:"token,omitempty"`
	TokenType  string `json:"token_type,omitempty"`
	TokenStore string `json:"token_store,omitempty"`
	TokenRef   string `json:"token_ref,omitempty"`
}

type Config struct {
	CurrentProfile string             `json:"current_profile"`
	Profiles       map[string]Profile `json:"profiles"`
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".omni", "config.json"), nil
}

func Load(path string) (*Config, error) {
	if path == "" {
		return nil, errors.New("config path is required")
	}

	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Profiles: map[string]Profile{}}, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	for name, profile := range cfg.Profiles {
		cfg.Profiles[name] = normalizeProfile(profile)
	}
	return &cfg, nil
}

func Save(path string, cfg *Config) error {
	if cfg == nil {
		return errors.New("config is nil")
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	for name, profile := range cfg.Profiles {
		cfg.Profiles[name] = normalizeProfile(profile)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("move temp config: %w", err)
	}

	return nil
}

func normalizeProfile(profile Profile) Profile {
	profile.BaseURL = strings.TrimSpace(profile.BaseURL)
	profile.DefaultAuth = normalizeAuthKind(profile.DefaultAuth)

	profile.PATStore = normalizeTokenStore(profile.PATStore)
	profile.PATToken = strings.TrimSpace(profile.PATToken)
	profile.PATRef = strings.TrimSpace(profile.PATRef)
	if profile.PATStore == "" && (profile.PATToken != "" || profile.PATRef != "") {
		profile.PATStore = defaultStoreForCredential(profile.PATToken, profile.PATRef)
	}

	profile.OrgKeyStore = normalizeTokenStore(profile.OrgKeyStore)
	profile.OrgKey = strings.TrimSpace(profile.OrgKey)
	profile.OrgKeyRef = strings.TrimSpace(profile.OrgKeyRef)
	if profile.OrgKeyStore == "" && (profile.OrgKey != "" || profile.OrgKeyRef != "") {
		profile.OrgKeyStore = defaultStoreForCredential(profile.OrgKey, profile.OrgKeyRef)
	}

	legacyType := normalizeAuthKind(profile.TokenType)
	legacyToken := strings.TrimSpace(profile.Token)
	legacyStore := normalizeTokenStore(profile.TokenStore)
	legacyRef := strings.TrimSpace(profile.TokenRef)
	if legacyStore == "" && (legacyToken != "" || legacyRef != "") {
		legacyStore = defaultStoreForCredential(legacyToken, legacyRef)
	}
	switch legacyType {
	case "org":
		if profile.OrgKey == "" && profile.OrgKeyRef == "" {
			profile.OrgKey = legacyToken
			profile.OrgKeyStore = legacyStore
			profile.OrgKeyRef = legacyRef
		}
	case "pat", "":
		if profile.PATToken == "" && profile.PATRef == "" {
			profile.PATToken = legacyToken
			profile.PATStore = legacyStore
			profile.PATRef = legacyRef
		}
	}

	if profile.DefaultAuth == "" {
		switch {
		case profile.HasPAT():
			profile.DefaultAuth = "pat"
		case profile.HasOrg():
			profile.DefaultAuth = "org"
		}
	}
	if profile.DefaultAuth == "pat" && !profile.HasPAT() && profile.HasOrg() {
		profile.DefaultAuth = "org"
	}
	if profile.DefaultAuth == "org" && !profile.HasOrg() && profile.HasPAT() {
		profile.DefaultAuth = "pat"
	}

	profile.Token = ""
	profile.TokenType = ""
	profile.TokenStore = ""
	profile.TokenRef = ""

	return profile
}

func (p Profile) HasPAT() bool {
	return strings.TrimSpace(p.PATToken) != "" || strings.TrimSpace(p.PATRef) != ""
}

func (p Profile) HasOrg() bool {
	return strings.TrimSpace(p.OrgKey) != "" || strings.TrimSpace(p.OrgKeyRef) != ""
}

func (p Profile) AuthMode() string {
	switch {
	case p.HasPAT() && p.HasOrg():
		return "both"
	case p.HasPAT():
		return "pat"
	case p.HasOrg():
		return "org"
	default:
		return ""
	}
}

func (p Profile) ConfiguredAuths() []string {
	auths := make([]string, 0, 2)
	if p.HasPAT() {
		auths = append(auths, "pat")
	}
	if p.HasOrg() {
		auths = append(auths, "org")
	}
	return auths
}

func normalizeAuthKind(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "pat":
		return "pat"
	case "org":
		return "org"
	default:
		return ""
	}
}

func normalizeTokenStore(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "config":
		return "config"
	case "keychain":
		return "keychain"
	default:
		return ""
	}
}

func defaultStoreForCredential(token string, ref string) string {
	if strings.TrimSpace(ref) != "" {
		return "keychain"
	}
	if strings.TrimSpace(token) != "" {
		return "config"
	}
	return ""
}
