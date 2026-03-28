package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// --- toSlug ---

func TestToSlug(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple", "My Project", "my-project"},
		{"uppercase", "HELLO WORLD", "hello-world"},
		{"special chars", "my_app@v2.0", "my-app-v2-0"},
		{"leading trailing spaces", "  test  ", "test"},
		{"multiple spaces", "a  b  c", "a-b-c"},
		{"empty string", "", "unnamed"},
		{"only special", "!!!@@@###", "unnamed"},
		{"hyphens preserved", "my-cool-project", "my-cool-project"},
		{"mixed", "Hello World -- Test!!!", "hello-world-test"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toSlug(tt.input)
			if got != tt.want {
				t.Errorf("toSlug(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// --- validateGitHubSignature ---

func TestValidateGitHubSignature_Valid(t *testing.T) {
	payload := []byte(`{"action":"push","ref":"refs/heads/main"}`)
	secret := "my-webhook-secret"

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if !validateGitHubSignature(payload, secret, sig) {
		t.Error("valid signature should pass")
	}
}

func TestValidateGitHubSignature_InvalidSignature(t *testing.T) {
	payload := []byte(`{"action":"push"}`)
	if validateGitHubSignature(payload, "secret", "sha256=deadbeef") {
		t.Error("invalid signature should fail")
	}
}

func TestValidateGitHubSignature_EmptyHeader(t *testing.T) {
	if validateGitHubSignature([]byte("data"), "secret", "") {
		t.Error("empty header should fail")
	}
}

func TestValidateGitHubSignature_MissingPrefix(t *testing.T) {
	if validateGitHubSignature([]byte("data"), "secret", "no-prefix-here") {
		t.Error("missing sha256= prefix should fail")
	}
}

func TestValidateGitHubSignature_WrongSecret(t *testing.T) {
	payload := []byte(`test payload`)
	secret := "correct-secret"

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if validateGitHubSignature(payload, "wrong-secret", sig) {
		t.Error("wrong secret should fail")
	}
}

// --- strPtrOrNil ---

func TestStrPtrOrNil_NonEmpty(t *testing.T) {
	got := strPtrOrNil("hello")
	if got == nil {
		t.Fatal("non-empty should return pointer")
	}
	if *got != "hello" {
		t.Errorf("*got = %q, want hello", *got)
	}
}

func TestStrPtrOrNil_Empty(t *testing.T) {
	got := strPtrOrNil("")
	if got != nil {
		t.Error("empty string should return nil")
	}
}

// --- parseLabelsString ---

func TestParseLabelsString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty", "", 0},
		{"empty object", "{}", 0},
		{"empty array", "[]", 0},
		{"single", "linux", 1},
		{"comma separated", "linux,docker,amd64", 3},
		{"json array", `["linux","docker"]`, 2},
		{"spaces", " linux , docker ", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLabelsString(tt.input)
			if len(got) != tt.want {
				t.Errorf("parseLabelsString(%q) = %d items, want %d: %v", tt.input, len(got), tt.want, got)
			}
		})
	}
}

func TestParseLabelsString_JsonArrayValues(t *testing.T) {
	labels := parseLabelsString(`["linux","docker","gpu"]`)
	if len(labels) != 3 {
		t.Fatalf("expected 3 labels, got %d: %v", len(labels), labels)
	}
	if labels[0] != "linux" {
		t.Errorf("labels[0] = %q, want linux", labels[0])
	}
	if labels[2] != "gpu" {
		t.Errorf("labels[2] = %q, want gpu", labels[2])
	}
}

// --- trimQuotesAndSpaces ---

func TestTrimQuotesAndSpaces(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`"hello"`, "hello"},
		{`  "hello"  `, "hello"},
		{"hello", "hello"},
		{`  spaces  `, "spaces"},
		{`""`, ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := trimQuotesAndSpaces(tt.input)
		if got != tt.want {
			t.Errorf("trimQuotesAndSpaces(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- parseIntOrDefault ---

func TestParseIntOrDefault(t *testing.T) {
	tests := []struct {
		input      string
		defaultVal int
		want       int
	}{
		{"42", 0, 42},
		{"", 10, 10},
		{"invalid", 5, 5},
		{"0", 10, 0},
		{"-1", 0, -1},
	}
	for _, tt := range tests {
		got := parseIntOrDefault(tt.input, tt.defaultVal)
		if got != tt.want {
			t.Errorf("parseIntOrDefault(%q, %d) = %d, want %d", tt.input, tt.defaultVal, got, tt.want)
		}
	}
}

// --- nilIfEmpty ---

func TestNilIfEmpty_NonEmpty(t *testing.T) {
	got := nilIfEmpty("hello")
	if got == nil {
		t.Error("non-empty should not return nil")
	}
	if s, ok := got.(string); !ok || s != "hello" {
		t.Errorf("nilIfEmpty(hello) = %v", got)
	}
}

func TestNilIfEmpty_Empty(t *testing.T) {
	got := nilIfEmpty("")
	if got != nil {
		t.Error("empty should return nil")
	}
}

// --- getEncKey ---

func TestGetEncKey_Valid(t *testing.T) {
	// 32 bytes = 64 hex chars
	hexKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	key, err := getEncKey(hexKey)
	if err != nil {
		t.Fatal(err)
	}
	if len(key) != 32 {
		t.Errorf("key length = %d, want 32", len(key))
	}
}

func TestGetEncKey_Empty(t *testing.T) {
	_, err := getEncKey("")
	if err == nil {
		t.Error("empty key should return error")
	}
}

func TestGetEncKey_TooShort(t *testing.T) {
	_, err := getEncKey("0123456789abcdef")
	if err == nil {
		t.Error("short key should return error")
	}
}

func TestGetEncKey_InvalidHex(t *testing.T) {
	_, err := getEncKey("zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz")
	if err == nil {
		t.Error("invalid hex should return error")
	}
}
