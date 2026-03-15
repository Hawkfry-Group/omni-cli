package cli

import (
	"fmt"
	"strings"

	"github.com/omni-co/omni-cli/internal/auth"
	"github.com/omni-co/omni-cli/internal/config"
	"github.com/omni-co/omni-cli/internal/secret"
)

func saveProfileWithCredentials(rt *runtime, profileName, baseURL, defaultAuth, patToken, orgKey, tokenStorePref string, setCurrent bool) (config.Profile, string, error) {
	store, err := secret.Pick(tokenStorePref, rt.Keychain)
	if err != nil {
		return config.Profile{}, "", err
	}

	newProfile := config.Profile{
		BaseURL:     strings.TrimSpace(baseURL),
		DefaultAuth: strings.TrimSpace(defaultAuth),
	}
	if token := strings.TrimSpace(patToken); token != "" {
		if err := applyCredential(store, tokenStorePref, profileName, "pat", token, &newProfile); err != nil {
			return config.Profile{}, "", err
		}
	}
	if token := strings.TrimSpace(orgKey); token != "" {
		if err := applyCredential(store, tokenStorePref, profileName, "org", token, &newProfile); err != nil {
			return config.Profile{}, "", err
		}
	}

	prev := rt.Config.Profiles[profileName]
	deleteRemovedCredentialRef(rt.Keychain, prev.PATStore, prev.PATRef, newProfile.PATStore, newProfile.PATRef)
	deleteRemovedCredentialRef(rt.Keychain, prev.OrgKeyStore, prev.OrgKeyRef, newProfile.OrgKeyStore, newProfile.OrgKeyRef)

	auth.SaveProfile(rt.Config, profileName, newProfile, setCurrent)
	if err := config.Save(rt.ConfigPath, rt.Config); err != nil {
		return config.Profile{}, "", err
	}

	warning := ""
	if tokenStorePref == "auto" && rt.Keychain != nil && !rt.Keychain.Available() {
		warning = "keychain unavailable; tokens stored in config file"
	}
	return rt.Config.Profiles[profileName], warning, nil
}

func applyCredential(store secret.Store, tokenStorePref, profileName, authKind, token string, profile *config.Profile) error {
	if store.Name() == "keychain" {
		refName := credentialRefName(profileName, authKind)
		ref, err := store.Save(refName, token)
		if err != nil {
			return fmt.Errorf("save %s credential in keychain: %w", authKind, err)
		}
		switch authKind {
		case "org":
			profile.OrgKeyStore = "keychain"
			profile.OrgKeyRef = ref
			profile.OrgKey = ""
		default:
			profile.PATStore = "keychain"
			profile.PATRef = ref
			profile.PATToken = ""
		}
		return nil
	}

	switch authKind {
	case "org":
		profile.OrgKeyStore = "config"
		profile.OrgKey = token
		profile.OrgKeyRef = ""
	default:
		profile.PATStore = "config"
		profile.PATToken = token
		profile.PATRef = ""
	}
	return nil
}

func credentialRefName(profileName, authKind string) string {
	return strings.TrimSpace(profileName) + ":" + strings.TrimSpace(authKind)
}

func deleteRemovedCredentialRef(store secret.Store, previousStore, previousRef, newStore, newRef string) {
	if store == nil || previousStore != "keychain" || strings.TrimSpace(previousRef) == "" {
		return
	}
	if newStore == "keychain" && strings.TrimSpace(newRef) == strings.TrimSpace(previousRef) {
		return
	}
	_ = store.Delete(previousRef)
}
