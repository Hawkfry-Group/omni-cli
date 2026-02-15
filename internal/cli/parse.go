package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
)

func parseOptionalBool(v string) (*bool, error) {
	s := strings.ToLower(strings.TrimSpace(v))
	switch s {
	case "":
		return nil, nil
	case "true", "1", "yes", "y":
		b := true
		return &b, nil
	case "false", "0", "no", "n":
		b := false
		return &b, nil
	default:
		return nil, fmt.Errorf("invalid boolean %q; use true or false", v)
	}
}

func parseUUIDArg(v, field string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(v))
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid %s %q: %w", field, v, err)
	}
	return id, nil
}

func parseOptionalUUIDArg(v, field string) (*uuid.UUID, error) {
	s := strings.TrimSpace(v)
	if s == "" {
		return nil, nil
	}
	id, err := parseUUIDArg(s, field)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func readJSONFile(path string) ([]byte, error) {
	payload, err := os.ReadFile(strings.TrimSpace(path))
	if err != nil {
		return nil, err
	}
	return payload, nil
}
