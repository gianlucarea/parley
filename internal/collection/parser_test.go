package collection

import (
	"os"
	"path/filepath"
	"testing"
)

const sampleTOML = `
[[requests]]
name   = "Get User"
method = "GET"
url    = "{{BASE_URL}}/users/{{USER_ID}}"

[requests.auth]
type  = "bearer"
token = "{{API_TOKEN}}"

[requests.headers]
Accept = "application/json"

[[requests]]
name     = "Create User"
method   = "POST"
url      = "{{BASE_URL}}/users"
body     = '{"name":"alice"}'

[requests.auth]
type     = "basic"
username = "admin"
password = "secret"
`

func writeTempTOML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "users.toml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp toml: %v", err)
	}
	return path
}

func TestParseFile_Basic(t *testing.T) {
	path := writeTempTOML(t, sampleTOML)
	c, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if c.Name != "users" {
		t.Errorf("Name = %q, want %q", c.Name, "users")
	}
	if c.FilePath != path {
		t.Errorf("FilePath = %q, want %q", c.FilePath, path)
	}
	if len(c.Requests) != 2 {
		t.Fatalf("len(Requests) = %d, want 2", len(c.Requests))
	}
}

func TestParseFile_Request_Fields(t *testing.T) {
	path := writeTempTOML(t, sampleTOML)
	c, _ := ParseFile(path)
	r := c.Requests[0]

	if r.Name != "Get User" {
		t.Errorf("Name = %q", r.Name)
	}
	if r.Method != "GET" {
		t.Errorf("Method = %q", r.Method)
	}
	if r.URL != "{{BASE_URL}}/users/{{USER_ID}}" {
		t.Errorf("URL = %q", r.URL)
	}
	if r.Auth.Type != "bearer" {
		t.Errorf("Auth.Type = %q", r.Auth.Type)
	}
	if r.Auth.Token != "{{API_TOKEN}}" {
		t.Errorf("Auth.Token = %q", r.Auth.Token)
	}
	if r.Headers["Accept"] != "application/json" {
		t.Errorf("Headers[Accept] = %q", r.Headers["Accept"])
	}
}

func TestParseFile_BasicAuth(t *testing.T) {
	path := writeTempTOML(t, sampleTOML)
	c, _ := ParseFile(path)
	r := c.Requests[1]

	if r.Auth.Type != "basic" {
		t.Errorf("Auth.Type = %q", r.Auth.Type)
	}
	if r.Auth.Username != "admin" {
		t.Errorf("Auth.Username = %q", r.Auth.Username)
	}
}

func TestParseFile_InvalidPath(t *testing.T) {
	_, err := ParseFile("/nonexistent/path/foo.toml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestParseFile_InvalidTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.toml")
	_ = os.WriteFile(path, []byte("[[requests]\nbroken"), 0o600)
	_, err := ParseFile(path)
	if err == nil {
		t.Fatal("expected error for invalid TOML")
	}
}

func TestParseDir(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.toml", "b.toml"} {
		_ = os.WriteFile(filepath.Join(dir, name), []byte("[[requests]]\nname=\"x\"\nmethod=\"GET\"\nurl=\"/x\""), 0o600)
	}
	// a non-toml file that should be ignored
	_ = os.WriteFile(filepath.Join(dir, "readme.md"), []byte("ignore me"), 0o600)

	cols, err := ParseDir(dir)
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}
	if len(cols) != 2 {
		t.Errorf("len = %d, want 2", len(cols))
	}
}

func TestParseDir_Empty(t *testing.T) {
	cols, err := ParseDir(t.TempDir())
	if err != nil {
		t.Fatalf("ParseDir empty: %v", err)
	}
	if len(cols) != 0 {
		t.Errorf("expected empty slice, got %d", len(cols))
	}
}

func TestParseDir_MissingDir(t *testing.T) {
	_, err := ParseDir("/nonexistent/dir")
	if err == nil {
		t.Fatal("expected error for missing directory")
	}
}
