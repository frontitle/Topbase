package internal_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestDependencyDirection(t *testing.T) {
	t.Parallel()

	rules := []struct {
		directory string
		forbidden []string
	}{
		{
			directory: "core",
			forbidden: []string{
				"github.com/topbase/topbase/internal/app",
				"github.com/topbase/topbase/internal/adapters",
				"github.com/topbase/topbase/internal/platform",
			},
		},
		{
			directory: "app",
			forbidden: []string{
				"github.com/topbase/topbase/internal/adapters",
				"github.com/topbase/topbase/internal/platform",
			},
		},
	}

	for _, rule := range rules {
		rule := rule
		t.Run(rule.directory, func(t *testing.T) {
			t.Parallel()
			assertNoForbiddenImports(t, rule.directory, rule.forbidden)
		})
	}
}

func assertNoForbiddenImports(t *testing.T, root string, forbidden []string) {
	t.Helper()

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			t.Errorf("parse %s: %v", path, err)
			return nil
		}
		for _, spec := range file.Decls {
			declaration, ok := spec.(*ast.GenDecl)
			if !ok || declaration.Tok != token.IMPORT {
				continue
			}
			for _, item := range declaration.Specs {
				importSpec, ok := item.(*ast.ImportSpec)
				if !ok {
					continue
				}
				importPath, err := strconv.Unquote(importSpec.Path.Value)
				if err != nil {
					continue
				}
				for _, prefix := range forbidden {
					if importPath == prefix || strings.HasPrefix(importPath, prefix+"/") {
						t.Errorf("%s imports forbidden outer layer %s", path, importPath)
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
}
