package auth

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/omni-co/omni-cli/internal/config"
	"github.com/omni-co/omni-cli/internal/secret"
)

type Options struct {
	ProfileFlag   string
	URLFlag       string
	TokenFlag     string
	TokenTypeFlag string
	ConfigPath    string
	TokenStore    secret.Store
}

type Resolved struct {
	ProfileName string
	Profile     config.Profile
	ConfigPath  string
}

func Resolve(cfg *config.Config, opts Options) (*Resolved, error) {
	profileName := firstNonEmpty(opts.ProfileFlag, os.Getenv("OMNI_PROFILE"), cfg.CurrentProfile)
	if profileName == "" {
		profileName = "default"
	}

	profile := cfg.Profiles[profileName]

	profile.BaseURL = firstNonEmpty(opts.URLFlag, os.Getenv("OMNI_URL"), profile.BaseURL)
	tokenOverride := firstNonEmpty(opts.TokenFlag, os.Getenv("OMNI_TOKEN"))
	if tokenOverride != "" {
		profile.Token = tokenOverride
	}
	profile.TokenType = normalizeTokenType(firstNonEmpty(opts.TokenTypeFlag, os.Getenv("OMNI_TOKEN_TYPE"), profile.TokenType))
	profile.TokenStore = normalizeTokenStore(profile.TokenStore)

	if profile.TokenType == "" {
		profile.TokenType = "pat"
	}
	if profile.TokenStore == "" {
		profile.TokenStore = "config"
	}
	if profile.Token == "" && profile.TokenStore == "keychain" && profile.TokenRef != "" {
		if opts.TokenStore == nil || !opts.TokenStore.Available() {
			return nil, errors.New("token is configured in keychain but keychain store is unavailable")
		}
		token, err := opts.TokenStore.Get(profile.TokenRef)
		if err != nil {
			return nil, fmt.Errorf("read token from keychain: %w", err)
		}
		profile.Token = token
	}

	if profile.BaseURL == "" {
		return nil, errors.New("missing Omni URL; set with --url or OMNI_URL or save in profile")
	}
	if profile.Token == "" {
		return nil, errors.New("missing API token; set with --token or OMNI_TOKEN or save in profile")
	}

	return &Resolved{
		ProfileName: profileName,
		Profile:     profile,
		ConfigPath:  opts.ConfigPath,
	}, nil
}

func SaveProfile(cfg *config.Config, profileName string, profile config.Profile, setCurrent bool) {
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]config.Profile{}
	}
	profile.TokenType = normalizeTokenType(profile.TokenType)
	if profile.TokenType == "" {
		profile.TokenType = "pat"
	}
	profile.TokenStore = normalizeTokenStore(profile.TokenStore)
	if profile.TokenStore == "" {
		profile.TokenStore = "config"
	}
	cfg.Profiles[profileName] = profile
	if setCurrent {
		cfg.CurrentProfile = profileName
	}
}

func UseProfile(cfg *config.Config, profileName string) error {
	if _, ok := cfg.Profiles[profileName]; !ok {
		return fmt.Errorf("profile %q not found", profileName)
	}
	cfg.CurrentProfile = profileName
	return nil
}

func normalizeTokenType(v string) string {
	s := strings.ToLower(strings.TrimSpace(v))
	switch s {
	case "", "pat", "org":
		return s
	default:
		return ""
	}
}

func normalizeTokenStore(v string) string {
	s := strings.ToLower(strings.TrimSpace(v))
	switch s {
	case "", "config", "keychain":
		return s
	default:
		return ""
	}
}

func RedactToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
