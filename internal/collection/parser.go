package collection

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// ParseFile loads and decodes a single .toml collection file.
// The Collection's Name is derived from the filename (without extension).
func ParseFile(path string) (Collection, error) {
	var c Collection
	if _, err := toml.DecodeFile(path, &c); err != nil {
		return Collection{}, err
	}
	c.FilePath = path
	c.Name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return c, nil
}

// ParseDir loads all .toml files in dir (non-recursive) and returns them in
// lexical order. Returns an empty slice (not an error) when no .toml files
// are present.
func ParseDir(dir string) ([]Collection, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var collections []Collection
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".toml" {
			continue
		}
		c, err := ParseFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		collections = append(collections, c)
	}
	return collections, nil
}
