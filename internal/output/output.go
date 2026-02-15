package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

type ErrorEnvelope struct {
	Error ErrorDetail `json:"error"`
}

func JSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func Human(w io.Writer, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}

func Plain(w io.Writer, v any) error {
	norm, err := normalize(v)
	if err != nil {
		return err
	}

	if rows, ok := extractRows(norm); ok {
		return printRowsTSV(w, rows)
	}

	switch x := norm.(type) {
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if _, err := fmt.Fprintf(w, "%s\t%s\n", k, formatPlainCell(x[k])); err != nil {
				return err
			}
		}
		return nil
	default:
		_, err := fmt.Fprintln(w, formatPlainCell(norm))
		return err
	}
}

func Print(v any, asJSON bool, asPlain ...bool) error {
	plain := len(asPlain) > 0 && asPlain[0]
	if asJSON {
		return JSON(os.Stdout, v)
	}
	if plain {
		return Plain(os.Stdout, v)
	}
	return Human(os.Stdout, v)
}

func Errorf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func PrintError(asJSON bool, code, message string, details any) {
	if asJSON {
		_ = JSON(os.Stderr, ErrorEnvelope{
			Error: ErrorDetail{
				Code:    code,
				Message: message,
				Details: details,
			},
		})
		return
	}
	if details != nil {
		b, _ := json.Marshal(details)
		fmt.Fprintf(os.Stderr, "%s: %s (%s)\n", code, message, string(b))
		return
	}
	fmt.Fprintf(os.Stderr, "%s: %s\n", code, message)
}

func normalize(v any) (any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func extractRows(v any) ([]map[string]any, bool) {
	switch x := v.(type) {
	case []any:
		return toRows(x), true
	case map[string]any:
		if rows, ok := x["records"].([]any); ok {
			return toRows(rows), true
		}
		if rows, ok := x["profiles"].([]any); ok {
			return toRows(rows), true
		}
		if rows, ok := x["results"].([]any); ok {
			return toRows(rows), true
		}
		if len(x) == 1 {
			for _, val := range x {
				if rows, ok := val.([]any); ok {
					return toRows(rows), true
				}
			}
		}
	}
	return nil, false
}

func toRows(items []any) []map[string]any {
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]any); ok {
			rows = append(rows, m)
			continue
		}
		rows = append(rows, map[string]any{"value": item})
	}
	return rows
}

func printRowsTSV(w io.Writer, rows []map[string]any) error {
	if len(rows) == 0 {
		return nil
	}

	headerSet := make(map[string]struct{})
	for _, row := range rows {
		for key := range row {
			headerSet[key] = struct{}{}
		}
	}
	headers := make([]string, 0, len(headerSet))
	for key := range headerSet {
		headers = append(headers, key)
	}
	sort.Strings(headers)

	if _, err := fmt.Fprintln(w, strings.Join(headers, "\t")); err != nil {
		return err
	}

	for _, row := range rows {
		cells := make([]string, 0, len(headers))
		for _, key := range headers {
			cells = append(cells, escapePlainCell(formatPlainCell(row[key])))
		}
		if _, err := fmt.Fprintln(w, strings.Join(cells, "\t")); err != nil {
			return err
		}
	}
	return nil
}

func formatPlainCell(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	case float64:
		if x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	case []any, map[string]any:
		b, _ := json.Marshal(x)
		return string(b)
	default:
		b, _ := json.Marshal(x)
		return string(b)
	}
}

func escapePlainCell(v string) string {
	replacer := strings.NewReplacer(
		"\t", " ",
		"\n", " ",
		"\r", " ",
	)
	return replacer.Replace(v)
}
