package grammars

import (
	"fmt"
	"testing"

	"github.com/odvcencio/gotreesitter"
)

func dumpLangDigest(lang *gotreesitter.Language) string {
	s := fmt.Sprintf("ptr=%p StateCount=%d LargeStateCount=%d SymbolCount=%d TokenCount=%d ExtTokenCount=%d LexStates=%d KwLexStates=%d ParseTable=%d SmallParseTable=%d ParseActions=%d ExternalSymbols=%v InitialState=%d KwCaptureToken=%d",
		lang, lang.StateCount, lang.LargeStateCount, len(lang.SymbolNames), lang.TokenCount, lang.ExternalTokenCount,
		len(lang.LexStates), len(lang.KeywordLexStates), len(lang.ParseTable), len(lang.SmallParseTable),
		len(lang.ParseActions), lang.ExternalSymbols, lang.InitialState, lang.KeywordCaptureToken)
	return s
}

func TestTSXIsolation_A_Before(t *testing.T) {
	lang := TsxLanguage()
	t.Logf("Before AllLanguages:\n  %s", dumpLangDigest(lang))
	
	p := gotreesitter.NewParser(lang)
	tree, _ := p.Parse([]byte(`require("lodash")`))
	root := tree.RootNode()
	got := sexpr(root, lang)
	t.Logf("Parse: hasError=%v sexpr=%s", root.HasError(), got)
	if root.HasError() {
		t.Error("parse failed BEFORE AllLanguages")
	}
}

func TestTSXIsolation_B_After(t *testing.T) {
	entries := AllLanguages()
	t.Logf("AllLanguages: %d entries", len(entries))
	
	lang := TsxLanguage()
	t.Logf("After AllLanguages:\n  %s", dumpLangDigest(lang))
	
	p := gotreesitter.NewParser(lang)
	tree, _ := p.Parse([]byte(`require("lodash")`))
	root := tree.RootNode()
	got := sexpr(root, lang)
	t.Logf("Parse: hasError=%v sexpr=%s", root.HasError(), got)
	if root.HasError() {
		t.Error("parse failed AFTER AllLanguages")
	}
}
