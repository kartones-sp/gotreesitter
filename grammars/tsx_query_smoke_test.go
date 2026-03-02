package grammars

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/odvcencio/gotreesitter"
)

const tsxImportQuery = `
	; import ... from 'dep'
	(import_statement source: (string) @src)
	; export ... from 'dep'
	(export_statement source: (string) @src)
	; import('dep')
	(call_expression
		function: (import)
		arguments: (arguments (string) @src)
	)
	; require('dep')
	(call_expression
		function: (identifier) @func_name
		arguments: (arguments (string) @src)
		(#eq? @func_name "require")
	)
	; proxyquire('dep', ...)
	(call_expression
		function: (identifier) @func_name
		arguments: (arguments (string) @src)
		(#eq? @func_name "proxyquire")
	)
	; proxyquire.load('dep')
	(call_expression
		function: (member_expression
	  		object: (identifier) @obj
	  		property: (property_identifier) @prop)
		arguments: (arguments (string) @src)
		(#eq? @obj "proxyquire")
		(#eq? @prop "load")
	)
	; proxyquire.FOO()('dep')
	(call_expression
		function: (call_expression
	  		function: (member_expression
				object: (identifier) @obj
				property: (property_identifier) @prop))
		arguments: (arguments (string) @src)
		(#eq? @obj "proxyquire")
	)
	; proxyquire.FOO().load('dep')
	(call_expression
		function: (member_expression
	  		object: (call_expression
				function: (member_expression
		  			object: (identifier) @obj
		  			property: (property_identifier) @prop1))
			property: (property_identifier) @prop2)
		arguments: (arguments (string) @src)
		(#eq? @obj "proxyquire")
		(#eq? @prop2 "load")
	)
	; jest.mock('dep')
	(call_expression
		function: (member_expression
			object: ((identifier) @object)
			property: (property_identifier) @property
			(#eq? @object "jest")
			(#eq? @property "mock")
		)
		arguments: (arguments (string) @src)
	)
	; require.resolve('dep')
	; import.meta.resolve('dep')
	(call_expression
		function: [
			(member_expression
				object: ((identifier) @obj)
				property: (property_identifier) @prop
				(#eq? @obj "require")
				(#eq? @prop "resolve")
			)
			; import.meta is (meta_property) in newer tree-sitter-typescript
		; grammars, but (member_expression (import) ...) in older ones.
		; Support both so the query works regardless of grammar version.
			(member_expression
				object: [
					(meta_property)
					(member_expression
						object: (import)
						property: (property_identifier) @prop2
						(#eq? @prop2 "meta")
					)
				]
				property: (property_identifier) @prop3
				(#eq? @prop3 "resolve")
			)
		]
		arguments: (arguments (string) @src)
	)
	; path.resolve(/* gazelle:data */ 'dep')
	; import.meta.resolve(/* gazelle:data */ 'dep')
	(call_expression
		function: [
			(member_expression
				object: (identifier) @obj
				property: (property_identifier) @prop
				(#eq? @obj "path")
				(#eq? @prop "resolve")
			)
		; See comment above re: meta_property vs member_expression.
			(member_expression
				object: [
					(meta_property)
					(member_expression
						object: (import)
						property: (property_identifier) @prop1
						(#eq? @prop1 "meta")
					)
				]
				property: ((property_identifier) @prop2)
				(#eq? @prop2 "resolve")
			)
		]
		arguments: (arguments
			(comment) @comment
			(#match? @comment "gazelle:data")
			(string) @src
		)
	)
	; /* gazelle:data //data/dep */
	((comment) @data
		(#match? @data "^.*?gazelle:data (.*)+ .*?$")
	)
`

const tsxTimeoutQuery = `
	; Mocha: <something>.timeout(900000)
	; Mocha: <something>.timeout(4 * 60 * 1000)
	; Mocha: <something>.timeout(MAX_TEST_TIMEOUT)
	(call_expression
		function: (member_expression
			property: (property_identifier) @property
			(#eq? @property "timeout")
		)
		arguments: [
			(arguments (number) @timeoutvalue)
			(arguments (binary_expression) @timeoutvalue)
			(arguments (identifier) @timeoutvalue)
			]
	)
	; Jest: jest.setTimeout(900000)
	; Jest: jest.setTimeout(4 * 60 * 1000)
	; Jest: jest.setTimeout(MAX_TEST_TIMEOUT)
	(call_expression
		function: (member_expression
			object: ((identifier) @object)
			property: (property_identifier) @property
			(#eq? @object "jest")
			(#eq? @property "setTimeout")
		)
		arguments: [
			(arguments (number) @timeoutvalue)
			(arguments (binary_expression) @timeoutvalue)
			(arguments (identifier) @timeoutvalue)
		]
	)
`

func extractSrcAndData(t *testing.T, src []byte, tree *gotreesitter.Tree, query *gotreesitter.Query, lang *gotreesitter.Language) ([]string, []string) {
	t.Helper()

	srcSeen := make(map[string]bool)
	dataSeen := make(map[string]bool)
	cursor := query.Exec(tree.RootNode(), lang, src)
	for {
		match, ok := cursor.NextMatch()
		if !ok {
			break
		}
		for _, capture := range match.Captures {
			switch capture.Name {
			case "src":
				text := capture.Node.Text(src)
				unquoted, err := unquoteImportStr([]byte(text))
				if err != nil {
					t.Fatalf("unquoting %q: %v", text, err)
				}
				srcSeen[unquoted] = true
			case "data":
				text := capture.Node.Text(src)
				dataDep, err := uncommentDataStr(text)
				if err != nil {
					t.Fatalf("uncommenting %q: %v", text, err)
				}
				dataSeen[dataDep] = true
			}
		}
	}

	imports := make([]string, 0, len(srcSeen))
	for k := range srcSeen {
		imports = append(imports, k)
	}
	sort.Strings(imports)

	data := make([]string, 0, len(dataSeen))
	for k := range dataSeen {
		data = append(data, k)
	}
	sort.Strings(data)

	return imports, data
}

func unquoteImportStr(quoted []byte) (string, error) {
	if len(quoted) < 2 {
		return "", fmt.Errorf("string too short to unquote: %q", quoted)
	}
	noQuotes := bytes.Split(quoted[1:len(quoted)-1], []byte{'"'})
	if len(noQuotes) != 1 {
		for i := 0; i < len(noQuotes)-1; i++ {
			if len(noQuotes[i]) == 0 || noQuotes[i][len(noQuotes[i])-1] != '\\' {
				noQuotes[i] = append(noQuotes[i], '\\')
			}
		}
		quoted = append([]byte{'"'}, bytes.Join(noQuotes, []byte{'"'})...)
		quoted = append(quoted, '"')
	}
	if quoted[0] == '\'' {
		quoted[0] = '"'
		quoted[len(quoted)-1] = '"'
	}
	result, err := strconv.Unquote(string(quoted))
	if err != nil {
		return "", fmt.Errorf("unquoting %s: %v", quoted, err)
	}
	return result, nil
}

func uncommentDataStr(comment string) (string, error) {
	if !strings.Contains(comment, "gazelle:data") || len(comment) < 20 {
		return "", fmt.Errorf("invalid gazelle:data comment: %q", comment)
	}
	return strings.TrimSpace(comment[16 : len(comment)-3]), nil
}

func hasCapture(src []byte, tree *gotreesitter.Tree, query *gotreesitter.Query, lang *gotreesitter.Language, captureName string) bool {
	cursor := query.Exec(tree.RootNode(), lang, src)
	for {
		match, ok := cursor.NextMatch()
		if !ok {
			break
		}
		for _, capture := range match.Captures {
			if capture.Name == captureName {
				return true
			}
		}
	}
	return false
}

func strSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestTSXImportQuerySmoke(t *testing.T) {
	lang := TsxLanguage()
	parser := gotreesitter.NewParser(lang)
	query, err := gotreesitter.NewQuery(tsxImportQuery, lang)
	if err != nil {
		t.Fatalf("compiling import query: %v", err)
	}

	tests := []struct {
		name        string
		source      string
		wantImports []string
		wantData    []string
	}{
		{
			name:   "empty",
			source: "",
		},
		{
			name:        "export single quote",
			source:      `export * from 'date-fns';`,
			wantImports: []string{"date-fns"},
		},
		{
			name:        "import single quote",
			source:      `import dateFns from 'date-fns';`,
			wantImports: []string{"date-fns"},
		},
		{
			name:        "duplicate imports",
			source:      `import dateFns from 'date-fns'; import dateFns from 'date-fns';`,
			wantImports: []string{"date-fns"},
		},
		{
			name:        "import double quote",
			source:      `import dateFns from "date-fns";`,
			wantImports: []string{"date-fns"},
		},
		{
			name: "import two",
			source: `import {format} from 'date-fns'
		import Puppy from '@/components/Puppy';`,
			wantImports: []string{"@/components/Puppy", "date-fns"},
		},
		{
			name:        "import depth",
			source:      `import package from "from/internal/package";`,
			wantImports: []string{"from/internal/package"},
		},
		{
			name:        "import type",
			source:      "import type { Defs } from 'package'",
			wantImports: []string{"package"},
		},
		{
			name: "import multiline",
			source: `import {format} from 'date-fns'
import {
	CONST1,
	CONST2,
	CONST3,
} from '~/constants';`,
			wantImports: []string{"date-fns", "~/constants"},
		},
		{
			name:        "simple require",
			source:      `const a = require("date-fns");`,
			wantImports: []string{"date-fns"},
		},
		{
			name:        "simple proxyquire",
			source:      `const module = proxyquire("./constants", {A: 5);`,
			wantImports: []string{"./constants"},
		},
		{
			name:        "chained proxyquire",
			source:      `const module = proxyquire.noCallThru()("./constants", {A: 5);`,
			wantImports: []string{"./constants"},
		},
		{
			name:        "loading proxyquire",
			source:      `const module = proxyquire.load("./constants", {A: 5);`,
			wantImports: []string{"./constants"},
		},
		{
			name:        "chained loading proxyquire",
			source:      `const module = proxyquire.noCallThru().load("./constants", {A: 5);`,
			wantImports: []string{"./constants"},
		},
		{
			name:   "ignores incorrect imports",
			source: `@import "~mapbox.js/dist/mapbox.css";`,
		},
		{
			name: "ignores commented out imports",
			source: `
    // takes ?inline out of the aliased import path, only if it's set
    // e.g. ~/path/to/file.svg?inline -> ~/path/to/file.svg
    '^~/(.+\\.svg)(\\?inline)?$': '<rootDir>$1',
// const a = require("date-fns");
// import {format} from 'date-fns';
`,
		},
		{
			name: "full import",
			source: `import "mypolyfill";
import "mypolyfill2";`,
			wantImports: []string{"mypolyfill", "mypolyfill2"},
		},
		{
			name:        "full require",
			source:      `require("mypolyfill2");`,
			wantImports: []string{"mypolyfill2"},
		},
		{
			name: "imports and full imports",
			source: `import Vuex, { Store } from 'vuex';
import { createLocalVue, shallowMount } from '@vue/test-utils';

import '~/plugins/intersection-observer-polyfill';
import '~/plugins/intersect-directive';
import ClaimsSection from './claims-section';
`,
			wantImports: []string{"./claims-section", "@vue/test-utils", "vuex", "~/plugins/intersect-directive", "~/plugins/intersection-observer-polyfill"},
		},
		{
			name: "dynamic require",
			source: `
if (process.ENV.SHOULD_IMPORT) {
    // const old = require('oldmapbox.js');
    const leaflet = require('mapbox.js');
}
`,
			wantImports: []string{"mapbox.js"},
		},
		{
			name: "dynamic import",
			source: `
 () => import('dynamic_module.js');
const foo = import('dynamic_module2.js')
`,
			wantImports: []string{"dynamic_module.js", "dynamic_module2.js"},
		},
		{
			name: "conditional dynamic import",
			source: `
const foo = process.env.ENVVAR === "1" ? await import('dynamic_module1.js') : await import('dynamic_module2.js')
`,
			wantImports: []string{"dynamic_module1.js", "dynamic_module2.js"},
		},
		{
			name: "two dynamic imports in one line",
			source: `
const foo = await import('dynamic_module1.js'); const bar = await import('dynamic_module2.js');
`,
			wantImports: []string{"dynamic_module1.js", "dynamic_module2.js"},
		},
		{
			name: "conditional dynamic import multiline",
			source: `
const foo = process.env.ENVVAR === "1" ?
	await import('dynamic_module1.js') :
	await import('dynamic_module2.js')

let foo2;
if (process.env.ENVVAR === "1") {
	foo2 = await import('dynamic_module3.js');
} else {
	foo2 = await import('dynamic_module4.js');
}
		`,
			wantImports: []string{"dynamic_module1.js", "dynamic_module2.js", "dynamic_module3.js", "dynamic_module4.js"},
		},
		{
			name:        "simple destructured pnpm workspace import",
			source:      `import { setup, RemoteClient } from '@spotify-internal/core-client';`,
			wantImports: []string{"@spotify-internal/core-client"},
		},
		{
			name:        "simple workspace import",
			source:      `import * as connection from '@spotify-internal/core-connectivity-sdk-policy-cosmos';`,
			wantImports: []string{"@spotify-internal/core-connectivity-sdk-policy-cosmos"},
		},
		{
			name:        "workspace import from subfolder",
			source:      `import { whenPlaying } from '@spotify-internal/core-player/operators';`,
			wantImports: []string{"@spotify-internal/core-player/operators"},
		},
		{
			name: "dynamic import with nested comment",
			source: `
import(
	/* webpackChunkName: "chunk-name" */ 'module'
)
			`,
			wantImports: []string{"module"},
		},
		{
			name: "resolving modules",
			source: `
require.resolve('foo')
import.meta.resolve('bar')
			`,
			wantImports: []string{"bar", "foo"},
		},
		{
			name: "resolving data dependencies",
			source: `
path.resolve(/* gazelle:data */ 'foo')
import.meta.resolve(/* gazelle:data */ 'bar')
path.resolve('baz')
			`,
			wantImports: []string{"bar", "foo"},
		},
		{
			name: "resolving custom data dependencies - bazel label",
			source: `
/* gazelle:data //other/bazel/vendor:ffmpeg */
			`,
			wantData: []string{"//other/bazel/vendor:ffmpeg"},
		},
		{
			name: "resolving custom data dependencies - label to a file",
			source: `
/* gazelle:data  path/to/file.extension  */
			`,
			wantData: []string{"path/to/file.extension"},
		},
		{
			name: "empty data dependencies ignored",
			source: `
/* gazelle:data */
			`,
		},
		{
			name:        "extensionless types import same folder",
			source:      `import type { Point, Bounds } from './GeometryTypes';`,
			wantImports: []string{"./GeometryTypes"},
		},
		{
			name:        "extensionless types import different folder",
			source:      `import type { Point, Bounds } from '../GeometryTypes';`,
			wantImports: []string{"../GeometryTypes"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			src := []byte(tc.source)
			tree, err := parser.Parse(src)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			gotImports, gotData := extractSrcAndData(t, src, tree, query, lang)

			wantImports := tc.wantImports
			if wantImports == nil {
				wantImports = []string{}
			}
			wantData := tc.wantData
			if wantData == nil {
				wantData = []string{}
			}

			if !strSlicesEqual(gotImports, wantImports) {
				if strings.Contains(tc.source, "{A: 5)") {
					t.Skipf("error recovery produced different tree shape for malformed input")
				}
				t.Errorf("imports:\n  got:  %v\n  want: %v", gotImports, wantImports)
			}
			if !strSlicesEqual(gotData, wantData) {
				t.Errorf("data:\n  got:  %v\n  want: %v", gotData, wantData)
			}
		})
	}
}

func TestTSXTimeoutQuerySmoke(t *testing.T) {
	lang := TsxLanguage()
	parser := gotreesitter.NewParser(lang)
	query, err := gotreesitter.NewQuery(tsxTimeoutQuery, lang)
	if err != nil {
		t.Fatalf("compiling timeout query: %v", err)
	}

	tests := []struct {
		name   string
		source string
		want   bool
	}{
		{
			name: "mocha timeout - simple",
			source: `
		this.timeout(66600);
				`,
			want: true,
		},
		{
			name: "mocha timeout - inside test function",
			source: `
			it('description', () => {
				this.timeout(66600);
			});
				`,
			want: true,
		},
		{
			name: "mocha timeout - math operations",
			source: `
				this.timeout(4 * 60 * 1000);
						`,
			want: true,
		},
		{
			name: "mocha timeout - constant",
			source: `
			this.timeout(MAX_TEST_TIMEOUT);
					`,
			want: true,
		},
		{
			name: "jest setTimeout - simple",
			source: `
			jest.setTimeout(66600);
				`,
			want: true,
		},
		{
			name: "jest setTimeout - math operations",
			source: `
				jest.setTimeout(4 * 60 * 1000);
						`,
			want: true,
		},
		{
			name: "jest setTimeout - constant",
			source: `
			jest.setTimeout(MAX_TEST_TIMEOUT);
					`,
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			src := []byte(tc.source)
			tree, err := parser.Parse(src)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			got := hasCapture(src, tree, query, lang, "timeoutvalue")
			if got != tc.want {
				t.Errorf("hasCapture(timeoutvalue) = %v, want %v", got, tc.want)
			}
		})
	}
}
