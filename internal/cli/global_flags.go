package cli

import "strings"

var globalBoolFlags = map[string]struct{}{
	"json":     {},
	"plain":    {},
	"verbose":  {},
	"no-input": {},
}

var globalStringFlags = map[string]struct{}{
	"profile":    {},
	"url":        {},
	"token":      {},
	"token-type": {},
	"config":     {},
}

// normalizeArgsForGlobalFlags lets global flags appear before or after commands.
func normalizeArgsForGlobalFlags(args []string) []string {
	globals := make([]string, 0, len(args))
	remaining := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			remaining = append(remaining, args[i:]...)
			break
		}
		if !strings.HasPrefix(arg, "--") || arg == "-" {
			remaining = append(remaining, arg)
			continue
		}

		name, hasInlineValue := splitLongFlag(arg)
		if _, ok := globalBoolFlags[name]; ok {
			globals = append(globals, arg)
			continue
		}
		if _, ok := globalStringFlags[name]; ok {
			globals = append(globals, arg)
			if !hasInlineValue && i+1 < len(args) {
				globals = append(globals, args[i+1])
				i++
			}
			continue
		}

		remaining = append(remaining, arg)
	}

	out := make([]string, 0, len(args))
	out = append(out, globals...)
	out = append(out, remaining...)
	return out
}

func splitLongFlag(arg string) (string, bool) {
	raw := strings.TrimPrefix(strings.TrimSpace(arg), "--")
	if raw == "" {
		return "", false
	}
	parts := strings.SplitN(raw, "=", 2)
	name := strings.ToLower(strings.TrimSpace(parts[0]))
	return name, len(parts) == 2
}
