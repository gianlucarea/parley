package collection

// Collection represents a single .toml collection file.
type Collection struct {
	Name     string    // derived from filename (no extension)
	FilePath string    // absolute path to the .toml file
	Requests []Request `toml:"requests"`
}

// Request represents a single HTTP request entry within a collection.
type Request struct {
	Name    string            `toml:"name"`
	Method  string            `toml:"method"`
	URL     string            `toml:"url"`
	Headers map[string]string `toml:"headers"`
	Body    string            `toml:"body"`
	Auth    Auth              `toml:"auth"`
	Expect  Expect            `toml:"expect"`
}

// Auth holds authentication configuration for a request.
// Type is either "bearer" or "basic".
type Auth struct {
	Type     string `toml:"type"`
	Token    string `toml:"token"`    // bearer
	Username string `toml:"username"` // basic
	Password string `toml:"password"` // basic
}

// Expect defines optional response assertions for a request.
// A zero-value Expect (Status == 0, BodyContains == nil) means no assertions.
type Expect struct {
	Status       int      `toml:"status"`
	BodyContains []string `toml:"body_contains"`
}
