package cli

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/omni-co/omni-cli/internal/client"
	"github.com/omni-co/omni-cli/internal/output"
	"golang.org/x/term"
)

type setupDefaults struct {
	Profile string
	URL     string
}

func runSetup(rt *runtime, args []string, defaults setupDefaults) int {
	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	profileDefault := firstNonEmpty(defaults.Profile, "default")

	var profile string
	var url string
	var authMode string
	var defaultAuth string
	var patToken string
	var orgKey string
	var legacyToken string
	var legacyTokenType string
	var tokenStore string
	var nonInteractive bool
	var noValidate bool
	var setCurrent bool
	var timeoutSec int

	fs.StringVar(&profile, "profile", profileDefault, "Profile name")
	fs.StringVar(&url, "url", defaults.URL, "Omni instance URL")
	fs.StringVar(&authMode, "auth-mode", "", "Auth setup: pat, org, or both")
	fs.StringVar(&defaultAuth, "default-auth", "", "Default auth for general commands: pat or org")
	fs.StringVar(&orgKey, "org-key", "", "Org API key to save")
	fs.StringVar(&legacyToken, "token", "", "Legacy single token input")
	fs.StringVar(&legacyTokenType, "token-type", "", "Legacy token type: pat or org")
	fs.StringVar(&tokenStore, "token-store", "auto", "auto, keychain, or config")
	fs.BoolVar(&nonInteractive, "non-interactive", false, "Disable prompts; require all values via flags/env")
	fs.BoolVar(&noValidate, "no-validate", false, "Skip API validation step")
	fs.BoolVar(&setCurrent, "set-current", true, "Set profile as current")
	fs.IntVar(&timeoutSec, "timeout-seconds", 20, "Validation timeout in seconds")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printSetupUsage()
			return 0
		}
		return usageFail(rt, err.Error())
	}

	authMode, defaultAuth, patToken, orgKey = applyLegacySetupFlags(authMode, defaultAuth, patToken, orgKey, legacyToken, legacyTokenType)

	if rt.NoInput {
		nonInteractive = true
	}
	canPrompt := !nonInteractive && stdinIsTerminal()
	reader := bufio.NewReader(os.Stdin)

	if canPrompt {
		fmt.Fprintln(os.Stderr, "Omni CLI setup")
		fmt.Fprintln(os.Stderr, "Enter values or press Enter to accept defaults.")

		var err error
		profile, err = promptInput(reader, "Profile name", profile)
		if err != nil {
			return fail(rt, 1, codeUsageError, "setup prompt failed", map[string]any{"error": err.Error()})
		}
		url, err = promptInput(reader, "Omni URL", url)
		if err != nil {
			return fail(rt, 1, codeUsageError, "setup prompt failed", map[string]any{"error": err.Error()})
		}
		authMode, err = promptInput(reader, "Auth setup (pat|org|both)", firstNonEmpty(authMode, "pat"))
		if err != nil {
			return fail(rt, 1, codeUsageError, "setup prompt failed", map[string]any{"error": err.Error()})
		}
		authMode = normalizeSetupAuthMode(authMode)
		if authMode == "both" {
			defaultAuth, err = promptInput(reader, "Default auth for general commands (pat|org)", firstNonEmpty(defaultAuth, "pat"))
			if err != nil {
				return fail(rt, 1, codeUsageError, "setup prompt failed", map[string]any{"error": err.Error()})
			}
		}
		tokenStore, err = promptInput(reader, "Token store (auto|keychain|config)", tokenStore)
		if err != nil {
			return fail(rt, 1, codeUsageError, "setup prompt failed", map[string]any{"error": err.Error()})
		}
		if authMode == "org" || authMode == "both" {
			if strings.TrimSpace(orgKey) == "" {
				orgKey, err = promptSecretInput(reader, "Org API key")
				if err != nil {
					return fail(rt, 1, codeUsageError, "setup prompt failed", map[string]any{"error": err.Error()})
				}
			}
		}
	}

	profile = strings.TrimSpace(profile)
	url = strings.TrimSpace(url)
	authMode = normalizeSetupAuthMode(authMode)
	defaultAuth = normalizeSetupAuthKind(defaultAuth)
	patToken = strings.TrimSpace(patToken)
	orgKey = strings.TrimSpace(orgKey)
	tokenStore = normalizeTokenStoreSetting(tokenStore)

	if profile == "" {
		return usageFail(rt, "missing profile name; set --profile")
	}
	if url == "" {
		return usageFail(rt, "missing Omni URL; set --url")
	}
	if authMode == "" {
		return usageFail(rt, "missing auth setup; set --auth-mode to pat, org, or both")
	}
	if tokenStore == "" {
		return usageFail(rt, fmt.Sprintf("invalid token store %q; use auto, keychain, or config", tokenStore))
	}
	switch authMode {
	case "pat":
		defaultAuth = "pat"
	case "org":
		defaultAuth = "org"
		if orgKey == "" {
			return usageFail(rt, "missing org API key; set --org-key")
		}
	case "both":
		if orgKey == "" {
			return usageFail(rt, "missing org API key; set --org-key")
		}
		if defaultAuth == "" {
			return usageFail(rt, "missing default auth; set --default-auth to pat or org")
		}
	default:
		return usageFail(rt, fmt.Sprintf("invalid auth setup %q; use pat, org, or both", authMode))
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

	validated := false
	validationResults := map[string]validationSummary{}
	if !noValidate {
		if patToken != "" {
			summary, vErr := validateSetupCredential(url, patToken, "pat", timeoutSec)
			if vErr != nil {
				return fail(rt, 1, vErr.Code, vErr.Message, vErr.Details)
			}
			validationResults["pat"] = summary
			validated = validated || summary.Base.Status != "fail"
		}
		if orgKey != "" {
			summary, vErr := validateSetupCredential(url, orgKey, "org", timeoutSec)
			if vErr != nil {
				return fail(rt, 1, vErr.Code, vErr.Message, vErr.Details)
			}
			validationResults["org"] = summary
			validated = validated || summary.Base.Status != "fail"
		}
		emitSetupValidationMessages(authMode, validationResults)
	}

	profileCfg, warning, err := saveProfileWithCredentials(rt, profile, url, defaultAuth, patToken, orgKey, tokenStore, setCurrent)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to save profile", map[string]any{"error": err.Error()})
	}
	if warning != "" {
		output.Errorf("warning: %s", warning)
	}

	result := map[string]any{
		"ok":               true,
		"profile":          profile,
		"base_url":         url,
		"auth_mode":        profileCfg.AuthMode(),
		"default_auth":     profileCfg.DefaultAuth,
		"configured_auths": profileCfg.ConfiguredAuths(),
		"set_current":      setCurrent,
		"validated":        validated,
	}
	if note := setupValidationNote(authMode, validationResults); note != "" {
		result["note"] = note
	}
	if patToken != "" {
		result["pat_token_store"] = profileCfg.PATStore
	}
	if orgKey != "" {
		result["org_key_store"] = profileCfg.OrgKeyStore
	}
	if !noValidate {
		result["validation"] = validationResults
	}

	if err := output.Print(result, rt.JSON, rt.Plain); err != nil {
		return fail(rt, 1, codeAPIError, "failed to print output", map[string]any{"error": err.Error()})
	}
	return 0
}

func validateSetupCredential(url, token, authKind string, timeoutSec int) (validationSummary, *validationFailure) {
	api, err := client.New(url, token)
	if err != nil {
		return validationSummary{}, &validationFailure{Code: codeConfigError, Message: "failed to create API client", Details: map[string]any{"error": err.Error()}}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()
	return collectValidation(ctx, api, authKind, authKind == "org")
}

func emitSetupValidationMessages(authMode string, validationResults map[string]validationSummary) {
	patSummary, hasPAT := validationResults["pat"]
	orgSummary, hasOrg := validationResults["org"]

	if hasPAT && patSummary.Query.Status == "fail" {
		if authMode == "both" && hasOrg && orgSummary.Query.Status == "pass" && orgSummary.Admin.Status == "pass" {
			output.Errorf("note: PAT authenticated but has limited permissions on this instance (%s). Org key validation passed and setup succeeded.", patSummary.Query.Message)
		} else {
			output.Errorf("warning: PAT query capability validation failed (%s)", patSummary.Query.Message)
		}
	}
	if hasOrg && orgSummary.Query.Status == "fail" {
		output.Errorf("warning: org-key query capability validation failed (%s)", orgSummary.Query.Message)
	}
	if hasOrg && orgSummary.Admin.Status == "fail" {
		output.Errorf("warning: org-key admin capability validation failed (%s)", orgSummary.Admin.Message)
	}
}

func setupValidationNote(authMode string, validationResults map[string]validationSummary) string {
	if authMode != "both" {
		return ""
	}
	patSummary, hasPAT := validationResults["pat"]
	orgSummary, hasOrg := validationResults["org"]
	if !hasPAT || !hasOrg {
		return ""
	}
	if patSummary.Query.Status == "fail" && orgSummary.Query.Status == "pass" && orgSummary.Admin.Status == "pass" {
		return "PAT login worked, but this PAT has limited permissions on this instance. The org key passed full validation, so setup succeeded."
	}
	return ""
}

func applyLegacySetupFlags(authMode, defaultAuth, patToken, orgKey, token, tokenType string) (string, string, string, string) {
	token = strings.TrimSpace(token)
	tokenType = normalizeSetupAuthKind(tokenType)
	if token == "" || tokenType == "" {
		return authMode, defaultAuth, patToken, orgKey
	}
	switch tokenType {
	case "org":
		if strings.TrimSpace(orgKey) == "" {
			orgKey = token
		}
		if strings.TrimSpace(authMode) == "" {
			authMode = "org"
		}
		if strings.TrimSpace(defaultAuth) == "" {
			defaultAuth = "org"
		}
	default:
		if strings.TrimSpace(patToken) == "" {
			patToken = token
		}
		if strings.TrimSpace(authMode) == "" {
			authMode = "pat"
		}
		if strings.TrimSpace(defaultAuth) == "" {
			defaultAuth = "pat"
		}
	}
	return authMode, defaultAuth, patToken, orgKey
}

func printSetupUsage() {
	fmt.Print(`omni setup:
  omni setup [--profile <name>] [--url <url>] [--auth-mode <pat|org|both>]
             [--default-auth <pat|org>] [--org-key <key>]
             [--token-store auto|keychain|config] [--timeout-seconds 20]
  omni setup --non-interactive --profile <name> --url <url> --auth-mode both --org-key <key> --default-auth pat
  omni --no-input setup --profile <name> --url <url> --auth-mode org --org-key <key>
`)
}

func promptInput(reader *bufio.Reader, label, defaultValue string) (string, error) {
	if defaultValue != "" {
		fmt.Fprintf(os.Stderr, "%s [%s]: ", label, defaultValue)
	} else {
		fmt.Fprintf(os.Stderr, "%s: ", label)
	}
	text, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return strings.TrimSpace(defaultValue), nil
	}
	return text, nil
}

func promptSecretInput(reader *bufio.Reader, label string) (string, error) {
	fmt.Fprintf(os.Stderr, "%s: ", label)
	if stdinIsTerminal() {
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(b)), nil
	}
	text, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

func normalizeSetupAuthMode(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "pat":
		return "pat"
	case "org":
		return "org"
	case "both":
		return "both"
	default:
		return ""
	}
}

func normalizeSetupAuthKind(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "pat":
		return "pat"
	case "org":
		return "org"
	default:
		return ""
	}
}

func normalizeTokenStoreSetting(v string) string {
	s := strings.ToLower(strings.TrimSpace(v))
	switch s {
	case "", "auto":
		return "auto"
	case "keychain", "config":
		return s
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

func stdinIsTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
