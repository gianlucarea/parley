package env

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var varRE = regexp.MustCompile(`\{\{([A-Za-z_][A-Za-z0-9_]*)\}\}`)

// Load parses a .parley.env file at path and returns its key-value pairs.
// Lines beginning with '#' and blank lines are ignored.
// Values may optionally be wrapped in single or double quotes.
func Load(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	vars := make(map[string]string)
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx < 1 {
			return nil, fmt.Errorf("env line %d: invalid format, expected KEY=VALUE", lineNum)
		}
		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		// strip optional surrounding quotes
		if len(value) >= 2 &&
			((value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}
		vars[key] = value
	}
	return vars, scanner.Err()
}

// Substitute replaces every {{VAR}} placeholder in s.
// Resolution order: shell environment > fileVars.
// Returns an error listing any placeholders that could not be resolved.
func Substitute(s string, fileVars map[string]string) (string, error) {
	var missing []string
	result := varRE.ReplaceAllStringFunc(s, func(match string) string {
		name := match[2 : len(match)-2]
		if v, ok := os.LookupEnv(name); ok {
			return v
		}
		if v, ok := fileVars[name]; ok {
			return v
		}
		missing = append(missing, name)
		return match
	})
	if len(missing) > 0 {
		return result, fmt.Errorf("unresolved variables: %s", strings.Join(missing, ", "))
	}
	return result, nil
}

// Set holds the shared base variables (.parley.env) and any named
// environments (.parley.<name>.env) discovered in a directory.
type Set struct {
	Base  map[string]string
	Named map[string]map[string]string
	Names []string // sorted names of Named
}

// Discover loads .parley.env and every .parley.<name>.env file in dir.
// A missing base file is not an error.
func Discover(dir string) (Set, error) {
	set := Set{Base: map[string]string{}, Named: map[string]map[string]string{}}

	basePath := filepath.Join(dir, ".parley.env")
	base, err := Load(basePath)
	switch {
	case err == nil:
		set.Base = base
	case !errors.Is(err, os.ErrNotExist):
		return Set{}, fmt.Errorf("%s: %w", basePath, err)
	}

	matches, err := filepath.Glob(filepath.Join(dir, ".parley.*.env"))
	if err != nil {
		return Set{}, err
	}
	for _, path := range matches {
		file := filepath.Base(path)
		name := strings.TrimSuffix(strings.TrimPrefix(file, ".parley."), ".env")
		if name == "" {
			continue
		}
		vars, err := Load(path)
		if err != nil {
			return Set{}, fmt.Errorf("%s: %w", path, err)
		}
		set.Named[name] = vars
		set.Names = append(set.Names, name)
	}
	sort.Strings(set.Names)
	return set, nil
}

// Resolve returns the variables for the named environment: a copy of Base
// overlaid with that environment's values. An empty name returns Base only.
func (s Set) Resolve(name string) (map[string]string, error) {
	out := make(map[string]string, len(s.Base))
	for k, v := range s.Base {
		out[k] = v
	}
	if name == "" {
		return out, nil
	}
	named, ok := s.Named[name]
	if !ok {
		avail := "none"
		if len(s.Names) > 0 {
			avail = strings.Join(s.Names, ", ")
		}
		return nil, fmt.Errorf("unknown environment %q (available: %s)", name, avail)
	}
	for k, v := range named {
		out[k] = v
	}
	return out, nil
}
