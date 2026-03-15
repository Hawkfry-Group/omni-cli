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
	AuthFlag      string
	ConfigPath    string
	TokenStore    secret.Store
	RequireAuth   string
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

	stored := cfg.Profiles[profileName]
	profile := stored
	profile.BaseURL = firstNonEmpty(opts.URLFlag, os.Getenv("OMNI_URL"), profile.BaseURL)
	profile.DefaultAuth = normalizeAuthKind(profile.DefaultAuth)

	selectedAuth := resolveAuthKind(stored, opts)
	token, err := resolveToken(stored, selectedAuth, opts)
	if err != nil {
		return nil, err
	}

	if profile.BaseURL == "" {
		return nil, errors.New("missing Omni URL; set with --url or OMNI_URL or save in profile")
	}
	if token == "" {
		switch selectedAuth {
		case "org":
			return nil, errors.New("missing org API key; configure one with `omni setup` or `omni auth add`")
		default:
			return nil, errors.New("missing PAT; configure one with `omni setup` or `omni auth add`")
		}
	}

	profile.Token = token
	profile.TokenType = selectedAuth

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

	profile.BaseURL = strings.TrimSpace(profile.BaseURL)
	profile.DefaultAuth = normalizeAuthKind(profile.DefaultAuth)
	profile.PATStore = normalizeTokenStore(profile.PATStore)
	profile.PATToken = strings.TrimSpace(profile.PATToken)
	profile.PATRef = strings.TrimSpace(profile.PATRef)
	profile.OrgKeyStore = normalizeTokenStore(profile.OrgKeyStore)
	profile.OrgKey = strings.TrimSpace(profile.OrgKey)
	profile.OrgKeyRef = strings.TrimSpace(profile.OrgKeyRef)

	switch {
	case profile.DefaultAuth == "" && hasPAT(profile):
		profile.DefaultAuth = "pat"
	case profile.DefaultAuth == "" && hasOrg(profile):
		profile.DefaultAuth = "org"
	case profile.DefaultAuth == "pat" && !hasPAT(profile) && hasOrg(profile):
		profile.DefaultAuth = "org"
	case profile.DefaultAuth == "org" && !hasOrg(profile) && hasPAT(profile):
		profile.DefaultAuth = "pat"
	}

	profile.Token = ""
	profile.TokenType = ""
	profile.TokenStore = ""
	profile.TokenRef = ""

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

func RedactToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

func resolveAuthKind(profile config.Profile, opts Options) string {
	explicit := normalizeAuthKind(firstNonEmpty(opts.AuthFlag, opts.TokenTypeFlag, os.Getenv("OMNI_AUTH"), os.Getenv("OMNI_TOKEN_TYPE")))
	if explicit != "" {
		return explicit
	}

	required := normalizeAuthKind(opts.RequireAuth)
	if required != "" {
		return required
	}

	defaultAuth := normalizeAuthKind(profile.DefaultAuth)
	if defaultAuth != "" {
		return defaultAuth
	}
	if hasPAT(profile) {
		return "pat"
	}
	if hasOrg(profile) {
		return "org"
	}
	if strings.TrimSpace(os.Getenv("OMNI_PAT")) != "" {
		return "pat"
	}
	if strings.TrimSpace(os.Getenv("OMNI_ORG_KEY")) != "" {
		return "org"
	}
	return "pat"
}

func resolveToken(profile config.Profile, authKind string, opts Options) (string, error) {
	if token := firstNonEmpty(opts.TokenFlag, os.Getenv("OMNI_TOKEN")); token != "" {
		return token, nil
	}

	switch authKind {
	case "org":
		if token := strings.TrimSpace(os.Getenv("OMNI_ORG_KEY")); token != "" {
			return token, nil
		}
	default:
		if token := strings.TrimSpace(os.Getenv("OMNI_PAT")); token != "" {
			return token, nil
		}
	}

	token, storeName, ref := storedCredential(profile, authKind)
	if token != "" {
		return token, nil
	}
	if storeName == "keychain" && ref != "" {
		if opts.TokenStore == nil || !opts.TokenStore.Available() {
			return "", errors.New("token is configured in keychain but keychain store is unavailable")
		}
		keychainToken, err := opts.TokenStore.Get(ref)
		if err != nil {
			return "", fmt.Errorf("read token from keychain: %w", err)
		}
		return keychainToken, nil
	}

	return "", nil
}

func storedCredential(profile config.Profile, authKind string) (token string, store string, ref string) {
	switch authKind {
	case "org":
		if profile.OrgKey != "" || profile.OrgKeyRef != "" {
			return strings.TrimSpace(profile.OrgKey), normalizeTokenStore(profile.OrgKeyStore), strings.TrimSpace(profile.OrgKeyRef)
		}
	default:
		if profile.PATToken != "" || profile.PATRef != "" {
			return strings.TrimSpace(profile.PATToken), normalizeTokenStore(profile.PATStore), strings.TrimSpace(profile.PATRef)
		}
	}

	legacyType := normalizeAuthKind(profile.TokenType)
	if legacyType == "" {
		legacyType = "pat"
	}
	if legacyType != authKind {
		return "", "", ""
	}
	return strings.TrimSpace(profile.Token), normalizeTokenStore(profile.TokenStore), strings.TrimSpace(profile.TokenRef)
}

func hasPAT(profile config.Profile) bool {
	if strings.TrimSpace(profile.PATToken) != "" || strings.TrimSpace(profile.PATRef) != "" {
		return true
	}
	legacyType := normalizeAuthKind(profile.TokenType)
	return (legacyType == "" || legacyType == "pat") && (strings.TrimSpace(profile.Token) != "" || strings.TrimSpace(profile.TokenRef) != "")
}

func hasOrg(profile config.Profile) bool {
	if strings.TrimSpace(profile.OrgKey) != "" || strings.TrimSpace(profile.OrgKeyRef) != "" {
		return true
	}
	return normalizeAuthKind(profile.TokenType) == "org" && (strings.TrimSpace(profile.Token) != "" || strings.TrimSpace(profile.TokenRef) != "")
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

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
