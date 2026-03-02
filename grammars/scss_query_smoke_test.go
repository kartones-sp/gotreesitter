package grammars

import (
	"testing"

	"github.com/odvcencio/gotreesitter"
)

const scssImportQuery = `
	(import_statement (string_value) @src)
	(use_statement (string_value) @src)
	(forward_statement (string_value) @src)
`

func TestSCSSImportQuerySmoke(t *testing.T) {
	lang := ScssLanguage()
	parser := gotreesitter.NewParser(lang)
	query, err := gotreesitter.NewQuery(scssImportQuery, lang)
	if err != nil {
		t.Fatalf("compiling SCSS import query: %v", err)
	}

	tests := []struct {
		name        string
		source      string
		wantImports []string
	}{
		{
			name:   "empty file",
			source: "",
		},
		{
			name:        "import from a dependency",
			source:      `@import '@mock/encore-web/css/encore-light-theme';`,
			wantImports: []string{"@mock/encore-web/css/encore-light-theme"},
		},
		{
			name: "multiple dependency imports",
			source: `
				@import '@mock/encore-web/css/encore-light-theme';
				@import '@spotify-internal/something-else/css/sub-path';
			`,
			wantImports: []string{
				"@mock/encore-web/css/encore-light-theme",
				"@spotify-internal/something-else/css/sub-path",
			},
		},
		{
			name:        "import from a path",
			source:      `@import 'foo/bar';`,
			wantImports: []string{"foo/bar"},
		},
		{
			name:        "relative import",
			source:      `@import 'relative-import';`,
			wantImports: []string{"relative-import"},
		},
		{
			name:        "relative import with prefix",
			source:      `@import './relative-import';`,
			wantImports: []string{"./relative-import"},
		},
		{
			name:        "use import",
			source:      `@use "bootstrap";`,
			wantImports: []string{"bootstrap"},
		},
		{
			name:        "forward import",
			source:      `@forward "somewhere";`,
			wantImports: []string{"somewhere"},
		},
		{
			name: "multiple mixed type imports",
			source: `
				@import '@mock/encore-web/css/encore-light-theme';
				@import 'foo/bar';
				@import 'relative-import';
			`,
			wantImports: []string{
				"@mock/encore-web/css/encore-light-theme",
				"foo/bar",
				"relative-import",
			},
		},
		{
			name:        "built-in module",
			source:      `@use 'sass:map';`,
			wantImports: []string{"sass:map"},
		},
		{
			name:        "CSS module import without extension",
			source:      `@use 'SomeModule.module';`,
			wantImports: []string{"SomeModule.module"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			src := []byte(tc.source)
			tree, err := parser.Parse(src)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			gotImports, _ := extractSrcAndData(t, src, tree, query, lang)

			wantImports := tc.wantImports
			if wantImports == nil {
				wantImports = []string{}
			}

			if !strSlicesEqual(gotImports, wantImports) {
				t.Errorf("imports:\n  got:  %v\n  want: %v", gotImports, wantImports)
			}
		})
	}
}
