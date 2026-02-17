package cli

import (
	"reflect"
	"testing"
)

func TestNormalizeArgsForGlobalFlags(t *testing.T) {
	got := normalizeArgsForGlobalFlags([]string{"auth", "list", "--json"})
	want := []string{"--json", "auth", "list"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected normalized args:\nwant %#v\ngot  %#v", want, got)
	}
}

func TestNormalizeArgsForGlobalFlagsStringValue(t *testing.T) {
	got := normalizeArgsForGlobalFlags([]string{"auth", "list", "--profile", "prod"})
	want := []string{"--profile", "prod", "auth", "list"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected normalized args:\nwant %#v\ngot  %#v", want, got)
	}
}

func TestNormalizeArgsForGlobalFlagsRespectsDoubleDash(t *testing.T) {
	got := normalizeArgsForGlobalFlags([]string{"api", "call", "--", "--json"})
	want := []string{"api", "call", "--", "--json"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected normalized args:\nwant %#v\ngot  %#v", want, got)
	}
}
