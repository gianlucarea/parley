package env

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
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
