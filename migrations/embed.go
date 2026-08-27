// Package migrations exposes the immutable application-database migrations
// embedded in the Topbase binary. Released SQL files must never be edited;
// schema changes are introduced by adding the next numbered file.
package migrations

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
)

//go:embed *.sql
var files embed.FS

type File struct {
	Version int
	Name    string
	SQL     string
}

func Files() ([]File, error) {
	names, err := fs.Glob(files, "*.sql")
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	items := make([]File, 0, len(names))
	for _, name := range names {
		prefix, _, ok := strings.Cut(name, "_")
		if !ok {
			return nil, fmt.Errorf("migration %q must start with a numeric version", name)
		}
		version, err := strconv.Atoi(prefix)
		if err != nil || version <= 0 {
			return nil, fmt.Errorf("migration %q has an invalid version", name)
		}
		raw, err := files.ReadFile(name)
		if err != nil {
			return nil, err
		}
		items = append(items, File{Version: version, Name: name, SQL: string(raw)})
	}
	for i := 1; i < len(items); i++ {
		if items[i].Version == items[i-1].Version {
			return nil, fmt.Errorf("duplicate migration version %d", items[i].Version)
		}
	}
	return items, nil
}
