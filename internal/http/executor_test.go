package http

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gianlucarea/parley/internal/collection"
)

func TestExecute_GET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	req := collection.Request{
		Method: "GET",
		URL:    srv.URL + "/test",
	}
	resp, err := Execute(req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if resp.Body != `{"ok":true}` {
		t.Errorf("unexpected body: %s", resp.Body)
	}
	if resp.Elapsed <= 0 {
		t.Error("elapsed should be positive")
	}
}

func TestExecute_POST_WithBody(t *testing.T) {
	var received string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 512)
		n, _ := r.Body.Read(buf)
		received = string(buf[:n])
		w.WriteHeader(201)
	}))
	defer srv.Close()

	req := collection.Request{
		Method: "POST",
		URL:    srv.URL + "/items",
		Body:   `{"name":"parley"}`,
	}
	resp, err := Execute(req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}
	if received != `{"name":"parley"}` {
		t.Errorf("unexpected body received by server: %s", received)
	}
}

func TestExecute_BearerAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("Authorization")
		if got != "Bearer mytoken" {
			t.Errorf("expected 'Bearer mytoken', got %q", got)
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	req := collection.Request{
		Method: "GET",
		URL:    srv.URL,
		Auth:   collection.Auth{Type: "bearer", Token: "mytoken"},
	}
	_, err := Execute(req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecute_BasicAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expected := "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass"))
		got := r.Header.Get("Authorization")
		if got != expected {
			t.Errorf("expected %q, got %q", expected, got)
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	req := collection.Request{
		Method: "GET",
		URL:    srv.URL,
		Auth:   collection.Auth{Type: "basic", Username: "user", Password: "pass"},
	}
	_, err := Execute(req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecute_VarSubstitution(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/42" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	req := collection.Request{
		Method: "GET",
		URL:    srv.URL + "/users/{{USER_ID}}",
	}
	vars := map[string]string{"USER_ID": "42"}
	_, err := Execute(req, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecute_BearerAuth_VarSubstitution(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("Authorization")
		if got != "Bearer secret" {
			t.Errorf("expected 'Bearer secret', got %q", got)
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	req := collection.Request{
		Method: "GET",
		URL:    srv.URL,
		Auth:   collection.Auth{Type: "bearer", Token: "{{API_TOKEN}}"},
	}
	vars := map[string]string{"API_TOKEN": "secret"}
	_, err := Execute(req, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecute_UnsupportedMethod(t *testing.T) {
	req := collection.Request{Method: "HEAD", URL: "http://example.com"}
	_, err := Execute(req, nil)
	if err == nil {
		t.Error("expected error for unsupported method HEAD")
	}
}

func TestExecute_Assertions_NoExpect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	req := collection.Request{Method: "GET", URL: srv.URL}
	resp, err := Execute(req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Assertions) != 0 {
		t.Errorf("expected no assertions, got %d", len(resp.Assertions))
	}
}

func TestExecute_Assertions_StatusPass(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	req := collection.Request{
		Method: "GET",
		URL:    srv.URL,
		Expect: collection.Expect{Status: 200},
	}
	resp, err := Execute(req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Assertions) != 1 {
		t.Fatalf("expected 1 assertion, got %d", len(resp.Assertions))
	}
	a := resp.Assertions[0]
	if !a.Pass {
		t.Errorf("expected assertion to pass")
	}
	if a.Got != "200" || a.Want != "200" {
		t.Errorf("unexpected Got/Want: %q / %q", a.Got, a.Want)
	}
}

func TestExecute_Assertions_StatusFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
	}))
	defer srv.Close()

	req := collection.Request{
		Method: "GET",
		URL:    srv.URL,
		Expect: collection.Expect{Status: 200},
	}
	resp, err := Execute(req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Assertions) != 1 {
		t.Fatalf("expected 1 assertion, got %d", len(resp.Assertions))
	}
	a := resp.Assertions[0]
	if a.Pass {
		t.Errorf("expected assertion to fail")
	}
	if a.Got != "201" || a.Want != "200" {
		t.Errorf("unexpected Got/Want: %q / %q", a.Got, a.Want)
	}
}

func TestExecute_Assertions_BodyContainsPass(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true,"userId":1}`))
	}))
	defer srv.Close()

	req := collection.Request{
		Method: "GET",
		URL:    srv.URL,
		Expect: collection.Expect{BodyContains: []string{"ok", "userId"}},
	}
	resp, err := Execute(req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Assertions) != 2 {
		t.Fatalf("expected 2 assertions, got %d", len(resp.Assertions))
	}
	for _, a := range resp.Assertions {
		if !a.Pass {
			t.Errorf("expected assertion %q to pass", a.Name)
		}
	}
}

func TestExecute_Assertions_BodyContainsFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	req := collection.Request{
		Method: "GET",
		URL:    srv.URL,
		Expect: collection.Expect{BodyContains: []string{"ok", "missing"}},
	}
	resp, err := Execute(req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Assertions) != 2 {
		t.Fatalf("expected 2 assertions, got %d", len(resp.Assertions))
	}
	if !resp.Assertions[0].Pass {
		t.Errorf("expected first assertion to pass")
	}
	if resp.Assertions[1].Pass {
		t.Errorf("expected second assertion to fail")
	}
	if resp.Assertions[1].Got != "not found" {
		t.Errorf("expected Got to be 'not found', got %q", resp.Assertions[1].Got)
	}
}

func TestExecute_UnsupportedAuthType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	req := collection.Request{
		Method: "GET",
		URL:    srv.URL,
		Auth:   collection.Auth{Type: "oauth2"},
	}
	_, err := Execute(req, nil)
	if err == nil {
		t.Error("expected error for unsupported auth type")
	}
}

func TestExecute_ResponseHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "value")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	req := collection.Request{Method: "GET", URL: srv.URL}
	resp, err := Execute(req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Headers["X-Custom"] != "value" {
		t.Errorf("expected header X-Custom=value, got %q", resp.Headers["X-Custom"])
	}
}
