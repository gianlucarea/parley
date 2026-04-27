package http

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gianlucarea/parley/internal/collection"
	"github.com/gianlucarea/parley/internal/env"
)

// Response holds the result of an executed HTTP request.
type Response struct {
	StatusCode int
	Status     string
	Headers    map[string]string
	Body       string
	Elapsed    time.Duration
}

// Execute performs the HTTP request described by req, substituting any
// {{VAR}} placeholders using vars. Supported methods: GET, POST, PUT,
// PATCH, DELETE.
func Execute(req collection.Request, vars map[string]string) (Response, error) {
	// Substitute placeholders in URL, body, and auth fields.
	resolvedURL, err := env.Substitute(req.URL, vars)
	if err != nil {
		return Response{}, fmt.Errorf("url substitution: %w", err)
	}

	resolvedBody, err := env.Substitute(req.Body, vars)
	if err != nil {
		return Response{}, fmt.Errorf("body substitution: %w", err)
	}

	method := strings.ToUpper(req.Method)
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE":
		// valid
	default:
		return Response{}, fmt.Errorf("unsupported method: %s", req.Method)
	}

	var bodyReader io.Reader
	if resolvedBody != "" {
		bodyReader = strings.NewReader(resolvedBody)
	}

	httpReq, err := http.NewRequest(method, resolvedURL, bodyReader)
	if err != nil {
		return Response{}, fmt.Errorf("building request: %w", err)
	}

	// Apply headers with substitution.
	for k, v := range req.Headers {
		resolved, err := env.Substitute(v, vars)
		if err != nil {
			return Response{}, fmt.Errorf("header %q substitution: %w", k, err)
		}
		httpReq.Header.Set(k, resolved)
	}

	// Apply auth — auth fields also support {{VAR}} substitution.
	if err := applyAuth(httpReq, req.Auth, vars); err != nil {
		return Response{}, fmt.Errorf("auth: %w", err)
	}

	client := &http.Client{}
	start := time.Now()
	resp, err := client.Do(httpReq)
	elapsed := time.Since(start)
	if err != nil {
		return Response{}, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, fmt.Errorf("reading response body: %w", err)
	}

	headers := make(map[string]string, len(resp.Header))
	for k, vals := range resp.Header {
		headers[k] = strings.Join(vals, ", ")
	}

	return Response{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Headers:    headers,
		Body:       string(rawBody),
		Elapsed:    elapsed,
	}, nil
}

// applyAuth sets the Authorization header based on the auth configuration.
func applyAuth(req *http.Request, auth collection.Auth, vars map[string]string) error {
	switch strings.ToLower(auth.Type) {
	case "bearer":
		token, err := env.Substitute(auth.Token, vars)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
	case "basic":
		username, err := env.Substitute(auth.Username, vars)
		if err != nil {
			return err
		}
		password, err := env.Substitute(auth.Password, vars)
		if err != nil {
			return err
		}
		encoded := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
		req.Header.Set("Authorization", "Basic "+encoded)
	case "":
		// no auth
	default:
		return fmt.Errorf("unsupported auth type: %s", auth.Type)
	}
	return nil
}
