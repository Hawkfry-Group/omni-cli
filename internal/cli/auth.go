package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/omni-co/omni-cli/internal/auth"
	"github.com/omni-co/omni-cli/internal/config"
	"github.com/omni-co/omni-cli/internal/output"
)

func runAuth(rt *runtime, args []string) int {
	if len(args) == 0 {
		printAuthUsage()
		return 0
	}
	if wantsSubcommandHelp(args) {
		printAuthUsage()
		return 0
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "add":
		return runAuthAdd(rt, subArgs)
	case "list":
		return runAuthList(rt)
	case "remove", "rm":
		return runAuthRemove(rt, subArgs)
	case "use":
		return runAuthUse(rt, subArgs)
	case "whoami", "show":
		return runAuthShow(rt, subArgs)
	default:
		printAuthUsage()
		return usageFail(rt, fmt.Sprintf("unknown auth subcommand: %s", sub))
	}
}

func runAuthAdd(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("auth add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var name, url, authMode, defaultAuth, patToken, orgKey, legacyToken, legacyTokenType, tokenStore string
	var setCurrent bool

	fs.StringVar(&name, "name", "default", "Profile name")
	fs.StringVar(&url, "url", "", "Omni instance URL")
	fs.StringVar(&authMode, "auth-mode", "", "Auth setup: pat, org, or both")
	fs.StringVar(&defaultAuth, "default-auth", "", "Default auth for general commands: pat or org")
	fs.StringVar(&orgKey, "org-key", "", "Org API key to save")
	fs.StringVar(&legacyToken, "token", "", "Legacy single token input")
	fs.StringVar(&legacyTokenType, "token-type", "", "Legacy token type: pat or org")
	fs.StringVar(&tokenStore, "token-store", "auto", "auto, keychain, or config")
	fs.BoolVar(&setCurrent, "set-current", true, "Set as current profile")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printAuthUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}

	authMode, defaultAuth, patToken, orgKey = applyLegacySetupFlags(authMode, defaultAuth, patToken, orgKey, legacyToken, legacyTokenType)
	authMode = normalizeSetupAuthMode(authMode)
	defaultAuth = normalizeSetupAuthKind(defaultAuth)

	if url == "" {
		return usageFail(rt, "--url is required")
	}
	if authMode == "" {
		return usageFail(rt, "--auth-mode is required (pat, org, or both)")
	}
	switch authMode {
	case "pat":
		defaultAuth = "pat"
	case "org":
		defaultAuth = "org"
		if strings.TrimSpace(orgKey) == "" {
			return usageFail(rt, "--org-key is required for auth-mode org")
		}
	case "both":
		if strings.TrimSpace(orgKey) == "" {
			return usageFail(rt, "--org-key is required for auth-mode both")
		}
		if defaultAuth == "" {
			return usageFail(rt, "--default-auth is required for auth-mode both")
		}
	}

	if authMode == "pat" || authMode == "both" {
		if strings.TrimSpace(patToken) == "" {
			token, err := obtainPAT(rt, url)
			if err != nil {
				return fail(rt, 1, codeAuthError, "PAT browser login failed", map[string]any{"error": err.Error(), "base_url": url})
			}
			patToken = token
		}
	}

	profile, warning, err := saveProfileWithCredentials(rt, name, url, defaultAuth, patToken, orgKey, tokenStore, setCurrent)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to save profile", map[string]any{"error": err.Error()})
	}
	if warning != "" {
		output.Errorf("warning: %s", warning)
	}

	if err := output.Print(map[string]any{
		"ok":           true,
		"profile":      name,
		"auth_mode":    profile.AuthMode(),
		"default_auth": profile.DefaultAuth,
		"pat_store":    profile.PATStore,
		"org_store":    profile.OrgKeyStore,
	}, rt.JSON, rt.Plain); err != nil {
		return fail(rt, 1, codeAPIError, "failed to print output", map[string]any{"error": err.Error()})
	}
	return 0
}

func runAuthList(rt *runtime) int {
	profiles := make([]map[string]any, 0, len(rt.Config.Profiles))
	names := make([]string, 0, len(rt.Config.Profiles))
	for name := range rt.Config.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		profile := rt.Config.Profiles[name]
		profiles = append(profiles, map[string]any{
			"name":             name,
			"base_url":         profile.BaseURL,
			"auth_mode":        profile.AuthMode(),
			"default_auth":     profile.DefaultAuth,
			"configured_auths": profile.ConfiguredAuths(),
			"pat_store":        profile.PATStore,
			"org_store":        profile.OrgKeyStore,
			"current":          rt.Config.CurrentProfile == name,
		})
	}
	if err := output.Print(map[string]any{"profiles": profiles}, rt.JSON, rt.Plain); err != nil {
		return fail(rt, 1, codeAPIError, "failed to print output", map[string]any{"error": err.Error()})
	}
	return 0
}

func runAuthUse(rt *runtime, args []string) int {
	if len(args) != 1 {
		return usageFail(rt, "usage: omni auth use <profile>")
	}
	if err := auth.UseProfile(rt.Config, args[0]); err != nil {
		return fail(rt, 1, codeConfigMissing, "profile not found", map[string]any{"profile": args[0]})
	}
	if err := saveConfig(rt); err != nil {
		return fail(rt, 1, codeConfigError, "failed to save config", map[string]any{"error": err.Error()})
	}
	if err := output.Print(map[string]any{"ok": true, "current_profile": args[0]}, rt.JSON, rt.Plain); err != nil {
		return fail(rt, 1, codeAPIError, "failed to print output", map[string]any{"error": err.Error()})
	}
	return 0
}

func runAuthRemove(rt *runtime, args []string) int {
	if len(args) != 1 {
		return usageFail(rt, "usage: omni auth remove <profile>")
	}
	name := args[0]
	profile, ok := rt.Config.Profiles[name]
	if !ok {
		return fail(rt, 1, codeConfigMissing, "profile not found", map[string]any{"profile": name})
	}

	deleteRemovedCredentialRef(rt.Keychain, profile.PATStore, profile.PATRef, "", "")
	deleteRemovedCredentialRef(rt.Keychain, profile.OrgKeyStore, profile.OrgKeyRef, "", "")
	delete(rt.Config.Profiles, name)
	if rt.Config.CurrentProfile == name {
		rt.Config.CurrentProfile = ""
		for candidate := range rt.Config.Profiles {
			rt.Config.CurrentProfile = candidate
			break
		}
	}

	if err := saveConfig(rt); err != nil {
		return fail(rt, 1, codeConfigError, "failed to save config", map[string]any{"error": err.Error()})
	}
	if err := output.Print(map[string]any{"ok": true, "removed_profile": name, "current_profile": rt.Config.CurrentProfile}, rt.JSON, rt.Plain); err != nil {
		return fail(rt, 1, codeAPIError, "failed to print output", map[string]any{"error": err.Error()})
	}
	return 0
}

func runAuthShow(rt *runtime, args []string) int {
	fs := flag.NewFlagSet("auth show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var profileName string
	fs.StringVar(&profileName, "profile", "", "Profile name")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printAuthUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}

	name := strings.TrimSpace(profileName)
	if name == "" {
		name = rt.Config.CurrentProfile
	}
	if name == "" {
		name = "default"
	}
	profile, ok := rt.Config.Profiles[name]
	if !ok {
		return fail(rt, 1, codeConfigMissing, "profile not found", map[string]any{"profile": name})
	}

	if err := output.Print(map[string]any{
		"profile":          name,
		"base_url":         profile.BaseURL,
		"auth_mode":        profile.AuthMode(),
		"default_auth":     profile.DefaultAuth,
		"configured_auths": profile.ConfiguredAuths(),
		"pat_store":        profile.PATStore,
		"org_store":        profile.OrgKeyStore,
		"pat_token":        redactConfiguredToken(profile.PATToken, profile.PATRef),
		"org_key":          redactConfiguredToken(profile.OrgKey, profile.OrgKeyRef),
	}, rt.JSON, rt.Plain); err != nil {
		return fail(rt, 1, codeAPIError, "failed to print output", map[string]any{"error": err.Error()})
	}
	return 0
}

func saveConfig(rt *runtime) error {
	return config.Save(rt.ConfigPath, rt.Config)
}

func printAuthUsage() {
	fmt.Print(`omni auth commands:
  omni auth add --name <profile> --url <url> --auth-mode <pat|org|both> [--default-auth <pat|org>] [--org-key <key>] [--token-store auto|keychain|config]
  omni auth list
  omni auth remove <profile>
  omni auth use <profile>
  omni auth show [--profile <name>]
  omni auth whoami
`)
}

func redactConfiguredToken(token string, ref string) string {
	if strings.TrimSpace(token) != "" {
		return auth.RedactToken(strings.TrimSpace(token))
	}
	if strings.TrimSpace(ref) != "" {
		return "stored-in-keychain"
	}
	return ""
}
