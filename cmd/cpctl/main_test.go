package main

import (
	"strings"
	"testing"

	"github.com/greenSheep999/cursor-proxy-oss/sdk"
)

func TestParseFlags_Defaults(t *testing.T) {
	// No positional, no extras.
	f, err := parseFlags(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if f.baseURL != "http://localhost:8317" {
		t.Fatalf("baseURL default = %q", f.baseURL)
	}
	if f.authChannel != sdk.AuthBearer {
		t.Fatalf("authChannel default = %v", f.authChannel)
	}
}

func TestParseFlags_UrlAndKey(t *testing.T) {
	f, err := parseFlags([]string{"-u", "http://x:9", "-k", "sk-abc"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if f.baseURL != "http://x:9" || f.apiKey != "sk-abc" {
		t.Fatalf("bad flags: %+v", f)
	}
}

func TestParseFlags_AuthChannel(t *testing.T) {
	cases := map[string]sdk.AuthChannel{
		"bearer":         sdk.AuthBearer,
		"x-api-key":      sdk.AuthAPIKey,
		"apikey":         sdk.AuthAPIKey,
		"x-goog":         sdk.AuthGoogAPIKey,
		"gemini":         sdk.AuthGoogAPIKey,
		"x-goog-api-key": sdk.AuthGoogAPIKey,
		"query":          sdk.AuthQueryKey,
	}
	for in, want := range cases {
		f, err := parseFlags([]string{"-a", in}, nil)
		if err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		if f.authChannel != want {
			t.Fatalf("%s: got %v want %v", in, f.authChannel, want)
		}
	}
}

func TestParseFlags_UnknownFlag(t *testing.T) {
	_, err := parseFlags([]string{"--bogus", "x"}, nil)
	if err == nil {
		t.Fatal("expected error on unknown flag")
	}
	if !strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("wrong error: %v", err)
	}
}

func TestParseFlags_ExtraFlag(t *testing.T) {
	f, err := parseFlags([]string{"-m", "gpt-5", "text"}, map[string]bool{"m": true})
	if err != nil {
		t.Fatal(err)
	}
	if f.extra["m"] != "gpt-5" {
		t.Fatalf("extra m = %q", f.extra["m"])
	}
	if len(f.positional) != 1 || f.positional[0] != "text" {
		t.Fatalf("positional = %+v", f.positional)
	}
}
