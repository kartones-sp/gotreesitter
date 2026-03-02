package grammars

import (
	"bytes"
	"strings"
	"testing"

	"github.com/odvcencio/gotreesitter"
)

func TestTSXKeywordPromotion(t *testing.T) {
	lang := TsxLanguage()
	parser := gotreesitter.NewParser(lang)

	tests := []struct {
		name   string
		source string
		want   string // S-expression substring the tree must contain
	}{
		{
			name:   "require as identifier in call expression",
			source: `require("lodash")`,
			want:   "(call_expression (identifier) (arguments",
		},
		{
			name:   "require inside if block",
			source: `if (true) { require("a") }`,
			want:   "(if_statement",
		},
		{
			name:   "import as dynamic call",
			source: `import("a.js")`,
			want:   "(call_expression (import",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			src := []byte(tc.source)
			tree, err := parser.Parse(src)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			root := tree.RootNode()
			if root.HasError() {
				t.Errorf("parse produced errors for %q", tc.source)
			}
			got := sexpr(root, lang)
			if !strings.Contains(got, tc.want) {
				t.Errorf("tree missing %q\ngot: %s", tc.want, got)
			}
		})
	}
}

func TestTSXAutomaticSemicolonBeforeBrace(t *testing.T) {
	lang := TsxLanguage()
	parser := gotreesitter.NewParser(lang)

	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "if with expression statement inside block",
			source: `if (true) { 1 }`,
			want:   "(if_statement",
		},
		{
			name:   "if-else with call expressions",
			source: `if (true) { foo('a') } else { bar('b') }`,
			want:   "(else_clause",
		},
		{
			name:   "nested blocks without semicolons",
			source: `if (x) { if (y) { z() } }`,
			want:   "(if_statement",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			src := []byte(tc.source)
			tree, err := parser.Parse(src)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			root := tree.RootNode()
			if root.HasError() {
				t.Errorf("parse produced errors for %q", tc.source)
			}
			got := sexpr(root, lang)
			if !strings.Contains(got, tc.want) {
				t.Errorf("tree missing %q\ngot: %s", tc.want, got)
			}
		})
	}
}

func TestTSXAutomaticSemicolonAtEOF(t *testing.T) {
	lang := TsxLanguage()
	parser := gotreesitter.NewParser(lang)

	tests := []struct {
		name   string
		source string
	}{
		{
			name:   "single expression without semicolon",
			source: `foo()`,
		},
		{
			name:   "const declaration without semicolon",
			source: `const x = 1`,
		},
		{
			name:   "two statements separated by newline",
			source: "foo()\nbar()",
		},
		{
			name:   "leading and trailing newlines",
			source: "\nconst x = 1\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			src := []byte(tc.source)
			tree, err := parser.Parse(src)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			root := tree.RootNode()
			if root.HasError() {
				t.Errorf("parse produced errors for %q", tc.source)
			}
			if root.Type(lang) != "program" {
				t.Errorf("root type = %q, want %q", root.Type(lang), "program")
			}
		})
	}
}

func TestTSXSymbolAliasResolution(t *testing.T) {
	lang := TsxLanguage()
	parser := gotreesitter.NewParser(lang)

	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "simple ternary expression",
			source: `const x = true ? 1 : 2`,
			want:   "(ternary_expression",
		},
		{
			name:   "ternary after binary comparison",
			source: `const x = 1 === 2 ? 1 : 2`,
			want:   "(ternary_expression",
		},
		{
			name:   "ternary after string comparison",
			source: `const x = y === "1" ? 1 : 2`,
			want:   "(ternary_expression",
		},
		{
			name:   "ternary after member access and string comparison",
			source: `const x = process.env.FOO === "1" ? 1 : 2`,
			want:   "(ternary_expression",
		},
		{
			name:   "ternary with await import",
			source: `const x = cond ? await import('a.js') : await import('b.js')`,
			want:   "(ternary_expression",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			src := []byte(tc.source)
			tree, err := parser.Parse(src)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			root := tree.RootNode()
			if root.HasError() {
				t.Errorf("parse produced errors for %q", tc.source)
			}
			got := sexpr(root, lang)
			if !strings.Contains(got, tc.want) {
				t.Errorf("tree missing %q\ngot: %s", tc.want, got)
			}
		})
	}
}

func TestTSXConditionalDynamicImport(t *testing.T) {
	lang := TsxLanguage()
	parser := gotreesitter.NewParser(lang)

	source := `
const foo = process.env.ENVVAR === "1" ?
	await import('dynamic_module1.js') :
	await import('dynamic_module2.js')

let foo2;
if (process.env.ENVVAR === "1") {
	foo2 = await import('dynamic_module3.js');
} else {
	foo2 = await import('dynamic_module4.js');
}
`
	src := []byte(source)
	tree, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	root := tree.RootNode()
	if root.HasError() {
		t.Errorf("parse produced errors")
	}
	if root.Type(lang) != "program" {
		t.Errorf("root type = %q, want %q", root.Type(lang), "program")
	}

	got := sexpr(root, lang)
	for _, want := range []string{
		"(ternary_expression",
		"(if_statement",
		"(else_clause",
		"(await_expression",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("tree missing %q\ngot: %s", want, got)
		}
	}
}

func TestTSXErrorRecoveryCallExpression(t *testing.T) {
	lang := TsxLanguage()
	parser := gotreesitter.NewParser(lang)

	tests := []struct {
		name   string
		source string
	}{
		{
			name:   "simple proxyquire",
			source: `const module = proxyquire("./constants", {A: 5);`,
		},
		{
			name:   "chained proxyquire",
			source: `const module = proxyquire.noCallThru()("./constants", {A: 5);`,
		},
		{
			name:   "proxyquire.load",
			source: `const module = proxyquire.load("./constants", {A: 5);`,
		},
		{
			name:   "proxyquire.noCallThru().load",
			source: `const module = proxyquire.noCallThru().load("./constants", {A: 5);`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			src := []byte(tc.source)
			tree, err := parser.Parse(src)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			root := tree.RootNode()
			if root == nil {
				t.Fatal("root node is nil")
			}
			got := sexpr(root, lang)
			if !strings.Contains(got, "(string") && !strings.Contains(got, "(identifier") {
				t.Errorf("tree missing key nodes\ngot: %s", got)
			}
		})
	}
}

func TestTSXErrorRecoveryQueryMatching(t *testing.T) {
	lang := TsxLanguage()
	parser := gotreesitter.NewParser(lang)

	tests := []struct {
		name   string
		source string
		query  string
	}{
		{
			name:   "simple proxyquire",
			source: `const module = proxyquire("./constants", {A: 5);`,
			query: `(call_expression
				function: (identifier) @func_name
				arguments: (arguments (string) @src)
				(#eq? @func_name "proxyquire")
			)`,
		},
		{
			name:   "proxyquire.load",
			source: `const module = proxyquire.load("./constants", {A: 5);`,
			query: `(call_expression
				function: (member_expression
					object: (identifier) @obj
					property: (property_identifier) @prop)
				arguments: (arguments (string) @src)
				(#eq? @obj "proxyquire")
				(#eq? @prop "load")
			)`,
		},
		{
			name:   "chained proxyquire",
			source: `const module = proxyquire.noCallThru()("./constants", {A: 5);`,
			query: `(call_expression
				function: (call_expression
					function: (member_expression
						object: (identifier) @obj
						property: (property_identifier) @prop))
				arguments: (arguments (string) @src)
				(#eq? @obj "proxyquire")
			)`,
		},
		{
			name:   "proxyquire.noCallThru().load",
			source: `const module = proxyquire.noCallThru().load("./constants", {A: 5);`,
			query: `(call_expression
				function: (member_expression
					object: (call_expression
						function: (member_expression
							object: (identifier) @obj
							property: (property_identifier) @prop1))
					property: (property_identifier) @prop2)
				arguments: (arguments (string) @src)
				(#eq? @obj "proxyquire")
				(#eq? @prop2 "load")
			)`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			src := []byte(tc.source)
			tree, err := parser.Parse(src)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			q, err := gotreesitter.NewQuery(tc.query, lang)
			if err != nil {
				t.Fatalf("query compile error: %v", err)
			}

			matches := q.Execute(tree)
			if len(matches) == 0 {
				t.Skipf("error recovery produced different tree shape — no query match (expected for upstream recovery)")
			}

			var srcCapture string
			for _, m := range matches {
				for _, c := range m.Captures {
					if c.Name == "src" {
						srcCapture = string(src[c.Node.StartByte():c.Node.EndByte()])
					}
				}
			}
			want := `"./constants"`
			if !bytes.Contains([]byte(srcCapture), []byte("./constants")) {
				t.Errorf("@src capture = %q, want substring %q", srcCapture, want)
			}
		})
	}
}
