//go:build grammar_set_core

package grammars

var coreLanguageSet = map[string]struct{}{
	"bash":       {},
	"c":          {},
	"cpp":        {},
	"css":        {},
	"go":         {},
	"html":       {},
	"java":       {},
	"javascript": {},
	"json":       {},
	"lua":        {},
	"php":        {},
	"python":     {},
	"rust":       {},
	"sql":        {},
	"toml":       {},
	"tsx":        {},
	"typescript": {},
	"yaml":       {},
}

func compileTimeLanguageEnabled(name string) bool {
	_, ok := coreLanguageSet[name]
	return ok
}
