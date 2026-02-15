package cli

import (
	"fmt"
	"strings"

	"github.com/omni-co/omni-cli/internal/auth"
	"github.com/omni-co/omni-cli/internal/config"
	"github.com/omni-co/omni-cli/internal/secret"
)

func saveProfileWithToken(rt *runtime, profileName, baseURL, tokenType, token, tokenStorePref string, setCurrent bool) (config.Profile, string, error) {
	store, err := secret.Pick(tokenStorePref, rt.Keychain)
	if err != nil {
		return config.Profile{}, "", err
	}

	newProfile := config.Profile{
		BaseURL:   strings.TrimSpace(baseURL),
		TokenType: strings.TrimSpace(tokenType),
	}
	warning := ""
	if store.Name() == "keychain" {
		ref, saveErr := store.Save(profileName, token)
		if saveErr != nil {
			return config.Profile{}, "", fmt.Errorf("save token in keychain: %w", saveErr)
		}
		newProfile.TokenStore = "keychain"
		newProfile.TokenRef = ref
		newProfile.Token = ""
	} else {
		newProfile.TokenStore = "config"
		newProfile.Token = token
		if tokenStorePref == "auto" && rt.Keychain != nil && !rt.Keychain.Available() {
			warning = "keychain unavailable; token stored in config file"
		}
	}

	prev := rt.Config.Profiles[profileName]
	if prev.TokenStore == "keychain" && prev.TokenRef != "" && newProfile.TokenStore != "keychain" && rt.Keychain != nil {
		_ = rt.Keychain.Delete(prev.TokenRef)
	}

	auth.SaveProfile(rt.Config, profileName, newProfile, setCurrent)
	if err := config.Save(rt.ConfigPath, rt.Config); err != nil {
		return config.Profile{}, "", err
	}

	return rt.Config.Profiles[profileName], warning, nil
}
