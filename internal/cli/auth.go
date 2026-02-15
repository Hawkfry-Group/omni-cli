package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"sort"

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

	var name, url, token, tokenType, tokenStore string
	var setCurrent bool

	fs.StringVar(&name, "name", "default", "Profile name")
	fs.StringVar(&url, "url", "", "Omni instance URL")
	fs.StringVar(&token, "token", "", "PAT or org API key")
	fs.StringVar(&tokenType, "token-type", "pat", "pat or org")
	fs.StringVar(&tokenStore, "token-store", "auto", "auto, keychain, or config")
	fs.BoolVar(&setCurrent, "set-current", true, "Set as current profile")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printAuthUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}

	if url == "" || token == "" {
		return usageFail(rt, "--url and --token are required")
	}

	profile, warning, err := saveProfileWithToken(rt, name, url, tokenType, token, tokenStore, setCurrent)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to save profile", map[string]any{"error": err.Error()})
	}
	if warning != "" {
		output.Errorf("warning: %s", warning)
	}

	if err := output.Print(map[string]any{
		"ok":          true,
		"profile":     name,
		"token_store": profile.TokenStore,
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
		store := profile.TokenStore
		if store == "" {
			store = "config"
		}
		profiles = append(profiles, map[string]any{
			"name":        name,
			"base_url":    profile.BaseURL,
			"token_type":  profile.TokenType,
			"token_store": store,
			"current":     rt.Config.CurrentProfile == name,
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

	if profile.TokenStore == "keychain" && profile.TokenRef != "" && rt.Keychain != nil {
		_ = rt.Keychain.Delete(profile.TokenRef)
	}
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

	var profile, url, token, tokenType string
	fs.StringVar(&profile, "profile", "", "Profile name")
	fs.StringVar(&url, "url", "", "Omni instance URL")
	fs.StringVar(&token, "token", "", "PAT or org API key")
	fs.StringVar(&tokenType, "token-type", "", "pat or org")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printAuthUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}

	resolved, err := auth.Resolve(rt.Config, auth.Options{
		ProfileFlag:   profile,
		URLFlag:       url,
		TokenFlag:     token,
		TokenTypeFlag: tokenType,
		ConfigPath:    rt.ConfigPath,
		TokenStore:    rt.Keychain,
	})
	if err != nil {
		return fail(rt, 1, codeAuthError, "failed to resolve auth profile", map[string]any{"error": err.Error()})
	}

	profileCfg := rt.Config.Profiles[resolved.ProfileName]
	if err := output.Print(map[string]any{
		"profile":     resolved.ProfileName,
		"base_url":    resolved.Profile.BaseURL,
		"token_type":  resolved.Profile.TokenType,
		"token_store": profileCfg.TokenStore,
		"token":       auth.RedactToken(resolved.Profile.Token),
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
  omni auth add --name <profile> --url <url> --token <token> [--token-type pat|org] [--token-store auto|keychain|config]
  omni auth list
  omni auth remove <profile>
  omni auth use <profile>
  omni auth show
  omni auth whoami
`)
}
