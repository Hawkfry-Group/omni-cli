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
	Profile   string
	URL       string
	Token     string
	TokenType string
}

func runSetup(rt *runtime, args []string, defaults setupDefaults) int {
	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	profileDefault := firstNonEmpty(defaults.Profile, "default")
	tokenTypeDefault := firstNonEmpty(defaults.TokenType, "pat")

	var profile string
	var url string
	var token string
	var tokenType string
	var tokenStore string
	var nonInteractive bool
	var noValidate bool
	var setCurrent bool
	var timeoutSec int

	fs.StringVar(&profile, "profile", profileDefault, "Profile name")
	fs.StringVar(&url, "url", defaults.URL, "Omni instance URL")
	fs.StringVar(&token, "token", defaults.Token, "PAT or org API key")
	fs.StringVar(&tokenType, "token-type", tokenTypeDefault, "pat or org")
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
		tokenType, err = promptInput(reader, "Token type (pat|org)", tokenType)
		if err != nil {
			return fail(rt, 1, codeUsageError, "setup prompt failed", map[string]any{"error": err.Error()})
		}
		tokenStore, err = promptInput(reader, "Token store (auto|keychain|config)", tokenStore)
		if err != nil {
			return fail(rt, 1, codeUsageError, "setup prompt failed", map[string]any{"error": err.Error()})
		}
		if strings.TrimSpace(token) == "" {
			token, err = promptSecretInput(reader, "Token")
			if err != nil {
				return fail(rt, 1, codeUsageError, "setup prompt failed", map[string]any{"error": err.Error()})
			}
		}
	}

	profile = strings.TrimSpace(profile)
	url = strings.TrimSpace(url)
	token = strings.TrimSpace(token)
	tokenType = normalizeSetupTokenType(tokenType)
	tokenStore = normalizeTokenStoreSetting(tokenStore)

	if profile == "" {
		return usageFail(rt, "missing profile name; set --profile")
	}
	if url == "" {
		return usageFail(rt, "missing Omni URL; set --url")
	}
	if token == "" {
		return usageFail(rt, "missing token; set --token")
	}
	if tokenType == "" {
		return usageFail(rt, fmt.Sprintf("invalid token type %q; use pat or org", tokenType))
	}
	if tokenStore == "" {
		return usageFail(rt, fmt.Sprintf("invalid token store %q; use auto, keychain, or config", tokenStore))
	}

	validated := false
	var summary validationSummary
	if !noValidate {
		api, err := client.New(url, token)
		if err != nil {
			return fail(rt, 1, codeConfigError, "failed to create API client", map[string]any{"error": err.Error()})
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
		defer cancel()

		summary, vErr := collectValidation(ctx, api, tokenType, tokenType == "org")
		if vErr != nil {
			return fail(rt, 1, vErr.Code, vErr.Message, vErr.Details)
		}
		validated = summary.Base.Status != "fail"
		if summary.Query.Status == "fail" {
			output.Errorf("warning: query capability validation failed (%s)", summary.Query.Message)
		}
		if summary.Admin.Status == "fail" {
			output.Errorf("warning: admin capability validation failed (%s)", summary.Admin.Message)
		}
	}

	profileCfg, warning, err := saveProfileWithToken(rt, profile, url, tokenType, token, tokenStore, setCurrent)
	if err != nil {
		return fail(rt, 1, codeConfigError, "failed to save profile", map[string]any{"error": err.Error()})
	}
	if warning != "" {
		output.Errorf("warning: %s", warning)
	}

	result := map[string]any{
		"ok":          true,
		"profile":     profile,
		"base_url":    url,
		"token_type":  tokenType,
		"token_store": profileCfg.TokenStore,
		"set_current": setCurrent,
		"validated":   validated,
	}
	if !noValidate {
		result["validation"] = summary
	}

	if err := output.Print(result, rt.JSON, rt.Plain); err != nil {
		return fail(rt, 1, codeAPIError, "failed to print output", map[string]any{"error": err.Error()})
	}
	return 0
}

func printSetupUsage() {
	fmt.Print(`omni setup:
  omni setup [--profile <name>] [--url <url>] [--token <token>] [--token-type pat|org]
             [--token-store auto|keychain|config] [--timeout-seconds 20]
  omni setup --non-interactive --profile <name> --url <url> --token <token> --token-type pat|org
  omni --no-input setup --profile <name> --url <url> --token <token> --token-type pat|org
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

func normalizeSetupTokenType(v string) string {
	s := strings.ToLower(strings.TrimSpace(v))
	switch s {
	case "", "pat":
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
