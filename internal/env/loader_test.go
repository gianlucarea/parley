package env

import (
	"os"
	"path/filepath"
	"testing"
)

func writeEnvFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".parley.env")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}
	return path
}

// ---------------------------------------------------------------------------
// Load
// ---------------------------------------------------------------------------

func TestLoad_Basic(t *testing.T) {
	path := writeEnvFile(t, "BASE_URL=https://api.example.com\nAPI_TOKEN=secret\n")
	vars, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if vars["BASE_URL"] != "https://api.example.com" {
		t.Errorf("BASE_URL = %q", vars["BASE_URL"])
	}
	if vars["API_TOKEN"] != "secret" {
		t.Errorf("API_TOKEN = %q", vars["API_TOKEN"])
	}
}

func TestLoad_CommentsAndBlanks(t *testing.T) {
	content := `
# this is a comment
BASE_URL=https://example.com

# another comment
TOKEN=abc
`
	vars, err := Load(writeEnvFile(t, content))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(vars) != 2 {
		t.Errorf("len = %d, want 2", len(vars))
	}
}

func TestLoad_QuotedValues(t *testing.T) {
	content := `
DOUBLE="hello world"
SINGLE='foo bar'
`
	vars, err := Load(writeEnvFile(t, content))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if vars["DOUBLE"] != "hello world" {
		t.Errorf("DOUBLE = %q", vars["DOUBLE"])
	}
	if vars["SINGLE"] != "foo bar" {
		t.Errorf("SINGLE = %q", vars["SINGLE"])
	}
}

func TestLoad_ValueWithEquals(t *testing.T) {
	// values that contain '=' should be preserved
	vars, err := Load(writeEnvFile(t, "TOKEN=abc=def\n"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if vars["TOKEN"] != "abc=def" {
		t.Errorf("TOKEN = %q", vars["TOKEN"])
	}
}

func TestLoad_InvalidLine(t *testing.T) {
	_, err := Load(writeEnvFile(t, "NOEQUALSSIGN\n"))
	if err == nil {
		t.Fatal("expected error for invalid line")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load("/nonexistent/.parley.env")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

// ---------------------------------------------------------------------------
// Substitute
// ---------------------------------------------------------------------------

func TestSubstitute_FromFileVars(t *testing.T) {
	vars := map[string]string{"BASE_URL": "https://api.example.com", "USER_ID": "42"}
	got, err := Substitute("{{BASE_URL}}/users/{{USER_ID}}", vars)
	if err != nil {
		t.Fatalf("Substitute: %v", err)
	}
	want := "https://api.example.com/users/42"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSubstitute_ShellEnvOverridesFileVars(t *testing.T) {
	t.Setenv("BASE_URL", "https://override.example.com")
	vars := map[string]string{"BASE_URL": "https://file.example.com"}
	got, err := Substitute("{{BASE_URL}}/path", vars)
	if err != nil {
		t.Fatalf("Substitute: %v", err)
	}
	if got != "https://override.example.com/path" {
		t.Errorf("got %q", got)
	}
}

func TestSubstitute_MissingVar(t *testing.T) {
	_, err := Substitute("{{MISSING_VAR}}", map[string]string{})
	if err == nil {
		t.Fatal("expected error for missing var")
	}
}

func TestSubstitute_MultipleVars_OneMissing(t *testing.T) {
	vars := map[string]string{"PRESENT": "ok"}
	result, err := Substitute("{{PRESENT}}-{{ABSENT}}", vars)
	if err == nil {
		t.Fatal("expected error")
	}
	// placeholder should be left intact in partial result
	if result == "" {
		t.Error("expected partial result, got empty string")
	}
}

func TestSubstitute_NoPlaceholders(t *testing.T) {
	got, err := Substitute("no placeholders here", map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "no placeholders here" {
		t.Errorf("got %q", got)
	}
}

func TestSubstitute_EmptyString(t *testing.T) {
	got, err := Substitute("", map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("got %q", got)
	}
}
