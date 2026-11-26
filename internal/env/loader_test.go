package env

import (
	"os"
	"path/filepath"
	"strings"
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

// ---------------------------------------------------------------------------
// Discover / Resolve
// ---------------------------------------------------------------------------

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func TestDiscover_BaseAndNamed(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".parley.env", "BASE_URL=https://base\nTOKEN=shared\n")
	writeFile(t, dir, ".parley.staging.env", "BASE_URL=https://staging\n")
	writeFile(t, dir, ".parley.dev.env", "BASE_URL=http://localhost\n")
	writeFile(t, dir, ".parley.env.example", "IGNORED=1\n")

	set, err := Discover(dir)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if got := strings.Join(set.Names, ","); got != "dev,staging" {
		t.Errorf("Names = %q, want dev,staging", got)
	}
	if set.Base["TOKEN"] != "shared" {
		t.Errorf("Base TOKEN = %q", set.Base["TOKEN"])
	}
}

func TestDiscover_MissingBase(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".parley.prod.env", "BASE_URL=https://prod\n")
	set, err := Discover(dir)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(set.Base) != 0 || len(set.Names) != 1 {
		t.Errorf("unexpected set: %+v", set)
	}
}

func TestDiscover_EmptyDir(t *testing.T) {
	set, err := Discover(t.TempDir())
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if set.Base == nil || len(set.Names) != 0 {
		t.Errorf("unexpected set: %+v", set)
	}
}

func TestDiscover_ParseErrorNamesFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".parley.bad.env", "NOEQUALS\n")
	_, err := Discover(dir)
	if err == nil || !strings.Contains(err.Error(), ".parley.bad.env") {
		t.Fatalf("expected error mentioning file, got %v", err)
	}
}

func TestResolve(t *testing.T) {
	set := Set{
		Base:  map[string]string{"BASE_URL": "https://base", "TOKEN": "shared"},
		Named: map[string]map[string]string{"dev": {"BASE_URL": "http://localhost"}},
		Names: []string{"dev"},
	}

	base, err := set.Resolve("")
	if err != nil || base["BASE_URL"] != "https://base" {
		t.Fatalf("Resolve(\"\") = %v, %v", base, err)
	}

	dev, err := set.Resolve("dev")
	if err != nil {
		t.Fatalf("Resolve(dev): %v", err)
	}
	if dev["BASE_URL"] != "http://localhost" || dev["TOKEN"] != "shared" {
		t.Errorf("dev = %v", dev)
	}
	if set.Base["BASE_URL"] != "https://base" {
		t.Error("Resolve mutated Base")
	}

	if _, err := set.Resolve("nope"); err == nil || !strings.Contains(err.Error(), "dev") {
		t.Errorf("expected unknown-env error listing dev, got %v", err)
	}
}
