package cli

import (
	"strings"
	"testing"
)

func TestCompletionContainsExpandedSubcommands(t *testing.T) {
	bash := bashCompletion()
	if !strings.Contains(bash, "documents") || !strings.Contains(bash, "permissions") {
		t.Fatalf("bash completion missing expanded documents commands: %q", bash)
	}
	if !strings.Contains(bash, "transfer-ownership") {
		t.Fatalf("bash completion missing schedules transfer-ownership: %q", bash)
	}
	if !strings.Contains(bash, "api") {
		t.Fatalf("bash completion missing api command: %q", bash)
	}
	if !strings.Contains(bash, "users") || !strings.Contains(bash, "scim") {
		t.Fatalf("bash completion missing users/scim commands: %q", bash)
	}
	if !strings.Contains(bash, "ai") || !strings.Contains(bash, "generate-query") || strings.Contains(bash, "route") {
		t.Fatalf("bash completion missing ai commands: %q", bash)
	}

	zsh := zshCompletion()
	if !strings.Contains(zsh, "documents command") || !strings.Contains(zsh, "permissions") {
		t.Fatalf("zsh completion missing expanded documents commands: %q", zsh)
	}
	if !strings.Contains(zsh, "folders command") || !strings.Contains(zsh, "permissions") {
		t.Fatalf("zsh completion missing expanded folders commands: %q", zsh)
	}
	if !strings.Contains(zsh, "api command") {
		t.Fatalf("zsh completion missing api command: %q", zsh)
	}
	if !strings.Contains(zsh, "users command") || !strings.Contains(zsh, "scim command") {
		t.Fatalf("zsh completion missing users/scim commands: %q", zsh)
	}
	if !strings.Contains(zsh, "ai command") {
		t.Fatalf("zsh completion missing ai command: %q", zsh)
	}
	if strings.Contains(zsh, "ai command' 'route ") || strings.Contains(zsh, "ai command' 'route'") {
		t.Fatalf("zsh completion should not include ai route: %q", zsh)
	}

	fish := fishCompletion()
	if !strings.Contains(fish, "documents") || !strings.Contains(fish, "permissions") {
		t.Fatalf("fish completion missing expanded documents commands: %q", fish)
	}
	if !strings.Contains(fish, "transfer-ownership") {
		t.Fatalf("fish completion missing schedules transfer-ownership: %q", fish)
	}
	if !strings.Contains(fish, "__fish_seen_subcommand_from api") {
		t.Fatalf("fish completion missing api command: %q", fish)
	}
	if !strings.Contains(fish, "__fish_seen_subcommand_from users") || !strings.Contains(fish, "__fish_seen_subcommand_from scim") {
		t.Fatalf("fish completion missing users/scim commands: %q", fish)
	}
	if !strings.Contains(fish, "__fish_seen_subcommand_from ai") {
		t.Fatalf("fish completion missing ai command: %q", fish)
	}
	if strings.Contains(fish, "__fish_seen_subcommand_from ai\" -a \"route ") || strings.Contains(fish, "__fish_seen_subcommand_from ai\" -a \"route\"") {
		t.Fatalf("fish completion should not include ai route: %q", fish)
	}
}
